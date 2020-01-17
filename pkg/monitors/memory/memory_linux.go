package memory

import (
	"time"

	"github.com/shirou/gopsutil/mem"
	"github.com/signalfx/golib/v3/datapoint"
)

func (m *Monitor) makeMemoryDatapoints(memInfo *mem.VirtualMemoryStat, dimensions map[string]string) []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		datapoint.New("memory.buffered", dimensions, datapoint.NewIntValue(int64(memInfo.Buffers)), datapoint.Gauge, time.Time{}),
		// for some reason gopsutil decided to add slab_reclaimable to cached which collectd does not
		datapoint.New("memory.cached", dimensions, datapoint.NewIntValue(int64(memInfo.Cached-memInfo.SReclaimable)), datapoint.Gauge, time.Time{}),
		datapoint.New("memory.slab_recl", dimensions, datapoint.NewIntValue(int64(memInfo.SReclaimable)), datapoint.Gauge, time.Time{}),
		datapoint.New("memory.slab_unrecl", dimensions, datapoint.NewIntValue(int64(memInfo.Slab-memInfo.SReclaimable)), datapoint.Gauge, time.Time{}),
		datapoint.New("memory.free", dimensions, datapoint.NewIntValue(int64(memInfo.Free)), datapoint.Gauge, time.Time{}),
	}
}
