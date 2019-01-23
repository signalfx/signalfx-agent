package disk

import (
	"context"
	"regexp"
	"runtime"
	"strings"
	"time"

	gopsutil "github.com/shirou/gopsutil/disk"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/gopsutilhelper"
	log "github.com/sirupsen/logrus"
)

var iOCounters = gopsutil.IOCounters

const monitorType = "disk"

// MONITOR(disk):
// This monitor reports metrics about free disk space.
//
//
// ```yaml
// monitors:
//  - type: disk
// ```

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

// TODO: make ProcFSPath a global config

// Config for this monitor
type Config struct {
	config.MonitorConfig `singleInstance:"true" acceptsEndpoints:"false"`
	// The path to the proc filesystem. Useful to override in containerized
	// environments.  (Does not apply to windows)
	ProcFSPath string `yaml:"procFSPath" default:"/proc"`

	// Which devices to include/exclude
	Disks []string `yaml:"disks" default:"[\"/^loop[0-9]+$/\", \"/^dm-[0-9]+$/\"]"`

	// If true, the disks selected by `disks` will be excluded and all others
	// included.
	IgnoreSelected *bool `yaml:"ignoreSelected"`
	ignoreRegex    []*regexp.Regexp
	ignoreString   map[string]struct{}
}

func (m *Monitor) processWindowsDatapoints(disk *gopsutil.IOCountersStat, dimensions map[string]string) {
	// Even though the struct fields say otherwise, on windows gopsutil
	// fills these with averages per read/write.  So these have to be
	// gauges instead of the usual cumulative counter
	// disk_ops.read
	m.Output.SendDatapoint(datapoint.New("disk_ops.avg_read", dimensions, datapoint.NewIntValue(int64(disk.ReadCount)), datapoint.Gauge, time.Time{}))

	// disk_ops.write
	m.Output.SendDatapoint(datapoint.New("disk_ops.avg_write", dimensions, datapoint.NewIntValue(int64(disk.WriteCount)), datapoint.Gauge, time.Time{}))

	// disk_octets.read
	m.Output.SendDatapoint(datapoint.New("disk_octets.avg_read", dimensions, datapoint.NewIntValue(int64(disk.ReadBytes)), datapoint.Gauge, time.Time{}))

	// disk_octets.write
	m.Output.SendDatapoint(datapoint.New("disk_octets.avg_write", dimensions, datapoint.NewIntValue(int64(disk.WriteBytes)), datapoint.Gauge, time.Time{}))

	// disk_merged.read - N/A
	// disk_merged.write - N/A

	// disk_time.read
	m.Output.SendDatapoint(datapoint.New("disk_time.avg_read", dimensions, datapoint.NewIntValue(int64(disk.ReadTime)), datapoint.Gauge, time.Time{}))

	// disk_time.write
	m.Output.SendDatapoint(datapoint.New("disk_time.avg_write", dimensions, datapoint.NewIntValue(int64(disk.WriteTime)), datapoint.Gauge, time.Time{}))
}

func (m *Monitor) processLinuxDatapoints(disk *gopsutil.IOCountersStat, dimensions map[string]string) {
	// disk_ops.read
	m.Output.SendDatapoint(datapoint.New("disk_ops.read", dimensions, datapoint.NewIntValue(int64(disk.ReadCount)), datapoint.Counter, time.Time{}))

	// disk_ops.write
	m.Output.SendDatapoint(datapoint.New("disk_ops.write", dimensions, datapoint.NewIntValue(int64(disk.WriteCount)), datapoint.Counter, time.Time{}))

	// disk_octets.read
	m.Output.SendDatapoint(datapoint.New("disk_octets.read", dimensions, datapoint.NewIntValue(int64(disk.ReadBytes)), datapoint.Counter, time.Time{}))

	// disk_octets.write
	m.Output.SendDatapoint(datapoint.New("disk_octets.write", dimensions, datapoint.NewIntValue(int64(disk.WriteBytes)), datapoint.Counter, time.Time{}))

	// disk_merged.read
	m.Output.SendDatapoint(datapoint.New("disk_merged.read", dimensions, datapoint.NewIntValue(int64(disk.MergedReadCount)), datapoint.Counter, time.Time{}))

	// disk_merged.write
	m.Output.SendDatapoint(datapoint.New("disk_merged.write", dimensions, datapoint.NewIntValue(int64(disk.MergedWriteCount)), datapoint.Counter, time.Time{}))

	// disk_time.read
	m.Output.SendDatapoint(datapoint.New("disk_time.read", dimensions, datapoint.NewIntValue(int64(disk.ReadTime)), datapoint.Counter, time.Time{}))

	// disk_time.write
	m.Output.SendDatapoint(datapoint.New("disk_time.write", dimensions, datapoint.NewIntValue(int64(disk.WriteTime)), datapoint.Counter, time.Time{}))
}

func shouldSkipDisk(conf *Config, disk *gopsutil.IOCountersStat) (shouldSkip bool) {
	_, match := conf.ignoreString[disk.Name]
	match = match || utils.FindMatchString(disk.Name, conf.ignoreRegex)
	if (match && *conf.IgnoreSelected) || (!match && !*conf.IgnoreSelected) {
		shouldSkip = true
	}
	return
}

// EmitDatapoints emits a set of memory datapoints
func (m *Monitor) emitDatapoints(conf *Config) {
	iocounts, err := iOCounters()
	if err != nil {
		logger.WithError(err).Warningf("failed to load io counters. if this message repeates frequently there may be a problem")
	}
	// var total uint64
	for key, disk := range iocounts {
		// handle selecting disk
		if shouldSkipDisk(conf, &disk) {
			logger.Debugf("skipping disk '%s'", disk.Name)
			continue
		}

		pluginInstance := strings.Replace(key, " ", "_", -1)
		dimensions := map[string]string{"plugin": monitorType, "plugin_instance": pluginInstance}
		if runtime.GOOS == "windows" {
			m.processWindowsDatapoints(&disk, dimensions)
		} else {
			m.processLinuxDatapoints(&disk, dimensions)
		}
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

	// convert array of strings and/or regexp to map of strings and array of regexp
	var errs []error
	conf.ignoreRegex, conf.ignoreString, errs = utils.RegexpStringsToRegexp(conf.Disks)
	for _, err := range errs {
		logger.Errorf(err.Error())
	}

	// default IgnoreSelected to true
	if conf.IgnoreSelected == nil {
		t := true
		conf.IgnoreSelected = &t
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
