package memory

import (
	"time"

	"github.com/shirou/gopsutil/mem"
	"github.com/signalfx/golib/v3/datapoint"
)

func (m *Monitor) makeMemoryDatapoints(memInfo *mem.VirtualMemoryStat, dimensions map[string]string) []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		datapoint.New("memory.active", dimensions, datapoint.NewIntValue(int64(memInfo.Active)), datapoint.Gauge, time.Time{}),
		datapoint.New("memory.inactive", dimensions, datapoint.NewIntValue(int64(memInfo.Inactive)), datapoint.Gauge, time.Time{}),
		datapoint.New("memory.wired", dimensions, datapoint.NewIntValue(int64(memInfo.Wired)), datapoint.Gauge, time.Time{}),
		datapoint.New("memory.free", dimensions, datapoint.NewIntValue(int64(memInfo.Free)), datapoint.Gauge, time.Time{}),
	}
}
