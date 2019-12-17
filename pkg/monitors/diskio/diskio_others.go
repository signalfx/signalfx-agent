// +build !windows

package diskio

import (
	"context"
	"strings"
	"time"

	gopsutil "github.com/shirou/gopsutil/disk"
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	"github.com/signalfx/signalfx-agent/pkg/utils/filter"
)

var iOCounters = gopsutil.IOCounters

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel func()
	conf   *Config
	filter *filter.OverridableStringFilter
}

func (m *Monitor) makeLinuxDatapoints(disk gopsutil.IOCountersStat, dimensions map[string]string) []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		datapoint.New("disk_ops.read", dimensions, datapoint.NewIntValue(int64(disk.ReadCount)), datapoint.Counter, time.Time{}),
		datapoint.New("disk_ops.write", dimensions, datapoint.NewIntValue(int64(disk.WriteCount)), datapoint.Counter, time.Time{}),
		datapoint.New("disk_octets.read", dimensions, datapoint.NewIntValue(int64(disk.ReadBytes)), datapoint.Counter, time.Time{}),
		datapoint.New("disk_octets.write", dimensions, datapoint.NewIntValue(int64(disk.WriteBytes)), datapoint.Counter, time.Time{}),
		datapoint.New("disk_merged.read", dimensions, datapoint.NewIntValue(int64(disk.MergedReadCount)), datapoint.Counter, time.Time{}),
		datapoint.New("disk_merged.write", dimensions, datapoint.NewIntValue(int64(disk.MergedWriteCount)), datapoint.Counter, time.Time{}),
		datapoint.New("disk_time.read", dimensions, datapoint.NewIntValue(int64(disk.ReadTime)), datapoint.Counter, time.Time{}),
		datapoint.New("disk_time.write", dimensions, datapoint.NewIntValue(int64(disk.WriteTime)), datapoint.Counter, time.Time{}),
	}
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

		m.Output.SendDatapoints(m.makeLinuxDatapoints(disk, map[string]string{"plugin": monitorType, "plugin_instance": pluginInstance, "disk": pluginInstance})...)
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
		m.filter, err = filter.NewOverridableStringFilter([]string{"*"})
		logger.Debugf("empty disk list defaulting to '*'")
	} else {
		m.filter, err = filter.NewOverridableStringFilter(conf.Disks)
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
