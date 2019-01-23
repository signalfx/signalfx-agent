package df

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"time"

	gopsutil "github.com/shirou/gopsutil/disk"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/gopsutilhelper"

	log "github.com/sirupsen/logrus"
)

const monitorType = "df"

var part = gopsutil.Partitions
var usage = gopsutil.Usage

// MONITOR(df):
// This monitor reports metrics about free disk space.
//
//
// ```yaml
// monitors:
//  - type: df
// ```

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

// TODO: make ProcFSPath a global config

// Config for this monitor
type Config struct {
	config.MonitorConfig `singleInstance:"true" acceptsEndpoints:"false"`
	// The path to the proc filesystem. Useful to override in containerized
	// environments.  (Does not apply to windows)
	ProcFSPath string `yaml:"procFSPath" default:"/proc"`

	// If true, the filesystems selected by `fsTypes` and `mountPoints` will be
	// excluded and all others included.
	IgnoreSelected *bool `yaml:"ignoreSelected"`

	// The filesystem types to include/exclude.
	FSTypes []string `yaml:"fsTypes" default:"[\"aufs\", \"overlay\", \"tmpfs\", \"proc\", \"sysfs\", \"nsfs\", \"cgroup\", \"devpts\", \"selinuxfs\", \"devtmpfs\", \"debugfs\", \"mqueue\", \"hugetlbfs\", \"securityfs\", \"pstore\", \"binfmt_misc\", \"autofs\"]"`

	// The mount paths to include/exclude, is interpreted as a regex if
	// surrounded by `/`.  Note that you need to include the full path as the
	// agent will see it, irrespective of the hostFSPath option.
	MountPoints       []string `yaml:"mountPoints" default:"[\"/^/var/lib/docker/containers/\", \"/^/var/lib/rkt/pods/\", \"/^/net//\", \"/^/smb//\"]"`
	ReportByDevice    bool     `yaml:"reportByDevice" default:"false"`
	ReportInodes      bool     `yaml:"reportInodes" default:"false"`
	fsTypes           map[string]bool
	mountPoints       []*regexp.Regexp
	stringMountPoints map[string]struct{}
}

func getPluginInstance(conf *Config, partition *gopsutil.PartitionStat) (pluginInstance string) {
	if conf.ReportByDevice {
		pluginInstance = strings.Replace(partition.Device, " ", "_", -1)
	} else {
		pluginInstance = strings.Replace(partition.Mountpoint, " ", "_", -1)
	}
	return pluginInstance
}

func shouldSkipFileSystem(conf *Config, partition *gopsutil.PartitionStat) (shouldSkip bool) {
	if _, ok := conf.fsTypes[partition.Fstype]; (ok && *conf.IgnoreSelected) || (!ok && !*conf.IgnoreSelected) {
		shouldSkip = true
	}
	return
}

func shouldSkipMountpoint(conf *Config, partition *gopsutil.PartitionStat) (shouldSkip bool) {
	// check for plain string match
	_, match := conf.stringMountPoints[partition.Mountpoint]
	// check for regex match
	match = match || utils.FindMatchString(partition.Mountpoint, conf.mountPoints)
	if (match && *conf.IgnoreSelected) || (!match && !*conf.IgnoreSelected) {
		shouldSkip = true
	}
	return
}

func calculateUtil(used float64, total float64) (percent float64, err error) {
	if total == 0 {
		percent = 0
	} else {
		percent = used / total * 100
	}

	if percent < 0 {
		err = fmt.Errorf("percent %v <= 0 total: %v used: %v", percent, used, total)
	}

	if percent > 100 {
		err = fmt.Errorf("percent %v > 100 total: %v used: %v ", percent, used, total)
	}
	return percent, err
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
func (m *Monitor) emitDatapoints(conf *Config) {
	partitions, err := part(true)
	if err != nil {
		logger.WithError(err).Errorf("failed to collect list of mountpoints")
	}
	var used uint64
	var total uint64
	for _, partition := range partitions {

		// handle selecting fs types
		if shouldSkipFileSystem(conf, &partition) {
			logger.Debugf("skipping mountpoint `%s` with fs type `%s`", partition.Mountpoint, partition.Fstype)
			continue
		}

		// handle selecting mountpoints
		if shouldSkipMountpoint(conf, &partition) {
			logger.Debugf("skipping mountpoint '%s'", partition.Mountpoint)
			continue
		}

		// if we can't collect usage stats about the mountpoint then skip it
		disk, err := usage(partition.Mountpoint)
		if err != nil {
			logger.WithError(err).Errorf("failed to collect usage for mountpiont '%s'", partition.Mountpoint)
			continue
		}

		// set plugin instance according to reportByDevice config
		pluginInstance := getPluginInstance(conf, &partition)

		// disk utilization
		m.Output.SendDatapoint(datapoint.New("disk.utilization", map[string]string{"plugin": constants.UtilizationMetricPluginName, "plugin_instance": pluginInstance}, datapoint.NewFloatValue(disk.UsedPercent), datapoint.Gauge, time.Time{}))

		dimensions := map[string]string{"plugin": monitorType, "plugin_instance": pluginInstance}
		m.reportDFComplex(dimensions, disk)

		// report
		m.reportPercentBytes(dimensions, disk)

		// inodes are not available on windows
		if runtime.GOOS != "windows" && conf.ReportInodes {
			m.reportInodeDatapoints(dimensions, disk)
		}

		// update totals
		used += disk.Used
		total += (disk.Used + disk.Free)
	}

	if total > 0 {
		diskSummary, err := calculateUtil(float64(used), float64(total))
		if err != nil {
			logger.WithError(err).Errorf("failed to calculate utilization data")
			return
		}
		m.Output.SendDatapoint(datapoint.New("disk.summary_utilization", map[string]string{"plugin": constants.UtilizationMetricPluginName}, datapoint.NewFloatValue(diskSummary), datapoint.Gauge, time.Time{}))
	}
}

// Configure is the main function of the monitor, it will report host metadata
// on a varied interval
func (m *Monitor) Configure(conf *Config) error {
	if runtime.GOOS != "windows" {
		logger.Warningf("'%s' monitor is in beta on this platform.  For production environments please use 'collectd/%s'.", monitorType, monitorType)
	}

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	// set env vars for gopsutil
	if err := gopsutilhelper.SetEnvVars(map[string]string{gopsutilhelper.HostProc: conf.ProcFSPath}); err != nil {
		return err
	}

	// convert fstypes array to map for quick lookup
	conf.fsTypes = utils.StringSliceToMap(conf.FSTypes)

	// set IgnoreSelected to true by default
	if conf.IgnoreSelected == nil {
		t := true
		conf.IgnoreSelected = &t
	}

	// convert array of strings and/or regexp to map of strings and array of regexp
	var errs []error
	conf.mountPoints, conf.stringMountPoints, errs = utils.RegexpStringsToRegexp(conf.MountPoints)
	for _, err := range errs {
		if err != nil {
			return err
		}
	}

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		m.emitDatapoints(conf)
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel func()
}
