// +build !windows

package diskio

import (
	"context"
	"strings"
	"time"

	gopsutil "github.com/shirou/gopsutil/disk"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/filter"
)

var iOCounters = gopsutil.IOCounters

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel func()
	conf   *Config
	filter *filter.ExhaustiveStringFilter
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
		if err == context.DeadlineExceeded {
			logger.WithField("debug", err).Debugf("failed to load io counters. if this message repeats frequently there may be a problem")
		} else {
			logger.WithError(err).Errorf("failed to load io counters. if this message repeats frequently there may be a problem")
		}
	}
	// var total uint64
	for key, disk := range iocounts {
		// skip it if the disk doesn't match
		if !m.filter.Matches(disk.Name) {
			logger.Debugf("skipping disk '%s'", disk.Name)
			continue
		}

		pluginInstance := strings.Replace(key, " ", "_", -1)
		m.processLinuxDatapoints(&disk, map[string]string{"plugin": monitorType, "plugin_instance": pluginInstance, "disk": pluginInstance})
	}
}

// Configure is the main function of the monitor, it will report host metadata
// on a varied interval
func (m *Monitor) Configure(conf *Config) error {
	logger.Warningf("'%s' monitor is in beta on this platform.  For production environments please use 'collectd/%s'.", monitorType, monitorType)

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
