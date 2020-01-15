// +build !windows

package diskio

import (
	"context"
	"strings"
	"time"

	"github.com/shirou/gopsutil/disk"
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/utils/filter"
)

var iOCounters = disk.IOCountersWithContext

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	conf   *Config
	filter *filter.OverridableStringFilter
}

func (m *Monitor) makeLinuxDatapoints(disk disk.IOCountersStat, dimensions map[string]string) []*datapoint.Datapoint {
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
func (m *Monitor) emitDatapoints(ctx context.Context) {
	iocounts, err := iOCounters(ctx)
	if err != nil {
		logger.WithError(err).Errorf("Failed to load disk io counters")
		return
	}
	// var total uint64
	for key, disk := range iocounts {
		// skip it if the disk doesn't match
		if !m.filter.Matches(disk.Name) {
			logger.Debugf("skipping disk '%s'", disk.Name)
			continue
		}

		diskName := strings.Replace(key, " ", "_", -1)

		m.Output.SendDatapoints(m.makeLinuxDatapoints(disk, map[string]string{"disk": diskName})...)
	}
}

// Configure is the main function of the monitor, it will report host metadata
// on a varied interval
func (m *Monitor) Configure(conf *Config) error {
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

	return nil
}

func (m *Monitor) Collect(ctx context.Context) error {
	time.Sleep(5 * time.Second)
	m.emitDatapoints(ctx)
	return nil
}
