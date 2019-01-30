package diskio

import (
	"context"
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
	log "github.com/sirupsen/logrus"
)

var iOCounters = gopsutil.IOCounters

const monitorType = "disk-io"

// MONITOR(disk-io):
// This monitor reports I/O metrics about disks.
//
// On Linux hosts, this monitor relies on the `/proc` filesystem.
// If the underlying host's `/proc` file system is mounted somewhere other than
// /proc please specify the path using the top level configuration `procPath`.
//
// ```yaml
// procPath: /proc
// monitors:
//  - type: disk-io
// ```

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `singleInstance:"false" acceptsEndpoints:"false"`

	// The devices to include/exclude. This is a
	// [filter set](https://github.com/signalfx/signalfx-agent/blob/master/docs/filtering.md#generic-filters).
	Disks []string `yaml:"disks" default:"[\"*\", \"!/^loop[0-9]+$/\", \"!/^dm-[0-9]+$/\"]"`
}

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel func()
	conf   *Config
	filter *filter.ExhaustiveStringFilter
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

// EmitDatapoints emits a set of memory datapoints
func (m *Monitor) emitDatapoints() {
	iocounts, err := iOCounters()
	if err != nil {
		logger.WithError(err).Warningf("failed to load io counters. if this message repeates frequently there may be a problem")
	}
	// var total uint64
	for key, disk := range iocounts {
		// skip it if the disk doesn't match
		if !m.filter.Matches(disk.Name) {
			logger.Debugf("skipping disk '%s'", disk.Name)
			continue
		}

		pluginInstance := strings.Replace(key, " ", "_", -1)
		dimensions := map[string]string{"plugin": monitorType, "plugin_instance": pluginInstance, "disk": pluginInstance}
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

	// save conf to monitor for convenience
	m.conf = conf

	// configure filters
	var err error
	if len(conf.Disks) == 0 {
		m.filter, err = filter.NewExhaustiveStringFilter([]string{"*"})
		logger.Debugf("empty disk list defaulting to '*'")
	} else {
		m.filter, err = filter.NewExhaustiveStringFilter(conf.Disks)
	}

	// return an error if we can't set the filter
	if err != nil {
		return err
	}

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		m.emitDatapoints()
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
