package filesystems

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	gopsutil "github.com/shirou/gopsutil/disk"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/filter"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `singleInstance:"false" acceptsEndpoints:"false"`

	// Path to the root of the host filesystem.  Useful when running in a
	// container and the host filesystem is mounted in some subdirectory under
	// /.
	HostFSPath string `yaml:"hostFSPath"`

	// The filesystem types to include/exclude.  This is an [overridable
	// set](https://docs.signalfx.com/en/latest/integrations/agent/filtering.html#overridable-filters).
	FSTypes []string `yaml:"fsTypes" default:"[\"*\", \"!aufs\", \"!overlay\", \"!tmpfs\", \"!proc\", \"!sysfs\", \"!nsfs\", \"!cgroup\", \"!devpts\", \"!selinuxfs\", \"!devtmpfs\", \"!debugfs\", \"!mqueue\", \"!hugetlbfs\", \"!securityfs\", \"!pstore\", \"!binfmt_misc\", \"!autofs\"]"`

	// The mount paths to include/exclude. This is an [overridable
	// set](https://docs.signalfx.com/en/latest/integrations/agent/filtering.html#overridable-filters).
	// NOTE: If you are using the hostFSPath option you should not include the
	// `/hostfs/` mount in the filter.
	MountPoints []string `yaml:"mountPoints" default:"[\"*\", \"!/^/var/lib/docker/containers/\", \"!/^/var/lib/rkt/pods/\", \"!/^/net//\", \"!/^/smb//\", \"!/^/tmp/scratch/\"]"`
	// (Linux Only) If true, then metrics will be reported about logical devices.
	IncludeLogical bool `yaml:"includeLogical" default:"false"`
	// If true, then metrics will report with their plugin_instance set to the
	// device's name instead of the mountpoint.
	ReportByDevice bool `yaml:"reportByDevice" default:"false"`
	// (Linux Only) If true metrics will be reported about inodes.
	ReportInodes bool `yaml:"reportInodes" default:"false"`
}

// Monitor for Utilization
type Monitor struct {
	Output      types.Output
	cancel      func()
	conf        *Config
	hostFSPath  string
	fsTypes     *filter.OverridableStringFilter
	mountPoints *filter.OverridableStringFilter
	logger      logrus.FieldLogger
}

// returns common dimensions map according to reportInodes configuration
func (m *Monitor) getCommonDimensions(partition *gopsutil.PartitionStat) map[string]string {
	dims := map[string]string{
		"mountpoint":      strings.Replace(partition.Mountpoint, " ", "_", -1),
		"device":          strings.Replace(partition.Device, " ", "_", -1),
		"plugin_instance": "",
	}
	// sanitize hostfs path in mountpoint
	if m.hostFSPath != "" {
		dims["mountpoint"] = strings.Replace(dims["mountpoint"], m.hostFSPath, "", 1)
	}
	if m.conf.ReportByDevice {
		dims["plugin_instance"] = dims["device"]
	} else {
		dims["plugin_instance"] = dims["mountpoint"]
	}

	return dims
}

func (m *Monitor) reportInodeDatapoints(dimensions map[string]string, disk *gopsutil.UsageStat) {
	m.Output.SendDatapoint(datapoint.New("df_inodes.free", dimensions, datapoint.NewIntValue(int64(disk.InodesFree)), datapoint.Gauge, time.Time{}))
	m.Output.SendDatapoint(datapoint.New("df_inodes.used", dimensions, datapoint.NewIntValue(int64(disk.InodesUsed)), datapoint.Gauge, time.Time{}))
	// TODO: implement df_inodes.reserved
	m.Output.SendDatapoint(datapoint.New("percent_inodes.free", dimensions, datapoint.NewIntValue(int64(100-disk.InodesUsedPercent)), datapoint.Gauge, time.Time{}))
	m.Output.SendDatapoint(datapoint.New("percent_inodes.used", dimensions, datapoint.NewIntValue(int64(disk.InodesUsedPercent)), datapoint.Gauge, time.Time{}))
	// TODO: implement percent_inodes.reserved
}

func (m *Monitor) reportDFComplex(dimensions map[string]string, disk *gopsutil.UsageStat) {
	m.Output.SendDatapoint(datapoint.New("df_complex.free", dimensions, datapoint.NewIntValue(int64(disk.Free)), datapoint.Gauge, time.Time{}))
	m.Output.SendDatapoint(datapoint.New("df_complex.used", dimensions, datapoint.NewIntValue(int64(disk.Used)), datapoint.Gauge, time.Time{}))
	// TODO: implement df_complex.reserved
}

func (m *Monitor) reportPercentBytes(dimensions map[string]string, disk *gopsutil.UsageStat) {
	m.Output.SendDatapoint(datapoint.New("percent_bytes.free", dimensions, datapoint.NewFloatValue(100-disk.UsedPercent), datapoint.Gauge, time.Time{}))
	m.Output.SendDatapoint(datapoint.New("percent_bytes.used", dimensions, datapoint.NewFloatValue(disk.UsedPercent), datapoint.Gauge, time.Time{}))
	// TODO: implement percent_bytes.reserved
}

// emitDatapoints emits a set of memory datapoints
func (m *Monitor) emitDatapoints() {
	partitions, err := gopsutil.Partitions(m.conf.IncludeLogical)
	if err != nil {
		if err == context.DeadlineExceeded {
			m.logger.WithField("debug", err).Debugf("failed to collect list of mountpoints")
		} else {
			m.logger.WithError(err).Errorf("failed to collect list of mountpoints")
		}
	}
	var used uint64
	var total uint64
	for i := range partitions {
		partition := partitions[i]

		// skip it if the filesystem doesn't match
		if !m.fsTypes.Matches(partition.Fstype) {
			m.logger.Debugf("skipping mountpoint `%s` with fs type `%s`", partition.Mountpoint, partition.Fstype)
			continue
		}

		var mount string
		if m.hostFSPath != "" {
			mount = strings.Replace(partition.Mountpoint, m.hostFSPath, "", 1)
		} else {
			mount = partition.Mountpoint
		}

		// skip it if the mountpoint doesn't match
		if !m.mountPoints.Matches(mount) {
			m.logger.Debugf("skipping mountpoint '%s'", partition.Mountpoint)
			continue
		}

		// if we can't collect usage stats about the mountpoint then skip it
		disk, err := gopsutil.Usage(partition.Mountpoint)
		if err != nil {
			if err == context.DeadlineExceeded {
				m.logger.WithField("debug", err).Debugf("failed to collect usage for mountpoint '%s'", partition.Mountpoint)
			} else {
				m.logger.WithError(err).Errorf("failed to collect usage for mountpoint '%s'", partition.Mountpoint)
			}
			continue
		}

		// get common dimensions according to reportByDevice config
		commonDims := m.getCommonDimensions(&partition)

		// disk utilization
		m.Output.SendDatapoint(datapoint.New("disk.utilization",
			utils.MergeStringMaps(map[string]string{"plugin": types.UtilizationMetricPluginName}, commonDims),
			datapoint.NewFloatValue(disk.UsedPercent),
			datapoint.Gauge,
			time.Time{}),
		)

		dimensions := utils.MergeStringMaps(map[string]string{"plugin": monitorType}, commonDims)
		m.reportDFComplex(dimensions, disk)

		// report
		m.reportPercentBytes(dimensions, disk)

		// inodes are not available on windows
		if runtime.GOOS != "windows" && m.conf.ReportInodes {
			m.reportInodeDatapoints(dimensions, disk)
		}

		// update totals
		used += disk.Used
		total += (disk.Used + disk.Free)
	}

	diskSummary, err := calculateUtil(float64(used), float64(total))
	if err != nil {
		m.logger.WithError(err).Errorf("failed to calculate utilization data")
		return
	}
	m.Output.SendDatapoint(datapoint.New("disk.summary_utilization", map[string]string{"plugin": types.UtilizationMetricPluginName}, datapoint.NewFloatValue(diskSummary), datapoint.Gauge, time.Time{}))
}

// Configure is the main function of the monitor, it will report host metadata
// on a varied interval
func (m *Monitor) Configure(conf *Config) error {
	m.logger = logrus.WithFields(log.Fields{"monitorType": monitorType})
	if runtime.GOOS != "windows" {
		m.logger.Warningf("'%s' monitor is in beta on this platform.  For production environments please use 'collectd/%s'.", monitorType, monitorType)
	}

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	// save conf to monitor for quick reference
	m.conf = conf

	// configure filters
	var err error
	if len(m.conf.FSTypes) == 0 {
		m.fsTypes, err = filter.NewOverridableStringFilter([]string{"*"})
		m.logger.Debugf("empty fsTypes list, defaulting to '*'")
	} else {
		m.fsTypes, err = filter.NewOverridableStringFilter(m.conf.FSTypes)
	}

	// return an error if we can't set the filter
	if err != nil {
		return err
	}

	// strip trailing / from HostFSPath so when we do string replacement later
	// we are left with a path starting at /
	m.hostFSPath = strings.TrimRight(m.conf.HostFSPath, "/")

	// configure filters
	if len(m.conf.MountPoints) == 0 {
		m.mountPoints, err = filter.NewOverridableStringFilter([]string{"*"})
		m.logger.Debugf("empty mountPoints list, defaulting to '*'")
	} else {
		m.mountPoints, err = filter.NewOverridableStringFilter(m.conf.MountPoints)
	}

	// return an error if we can't set the filter
	if err != nil {
		return err
	}

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		m.emitDatapoints()
	}, time.Duration(m.conf.IntervalSeconds)*time.Second)

	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

func calculateUtil(used float64, total float64) (percent float64, err error) {
	if total == 0 {
		percent = 0
	} else {
		percent = used / total * 100
	}

	if percent < 0 {
		err = fmt.Errorf("percent %v < 0 total: %v used: %v", percent, used, total)
	}

	if percent > 100 {
		err = fmt.Errorf("percent %v > 100 total: %v used: %v ", percent, used, total)
	}
	return percent, err
}
