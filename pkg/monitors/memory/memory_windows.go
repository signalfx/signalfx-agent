package memory

import (
	"time"

	"github.com/shirou/gopsutil/mem"
	"github.com/signalfx/golib/v3/datapoint"
)

func (m *Monitor) makeMemoryDatapoints(memInfo *mem.VirtualMemoryStat, dimensions map[string]string) []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		datapoint.New("memory.utilization", dimensions, datapoint.NewFloatValue(memInfo.UsedPercent), datapoint.Gauge, time.Time{}),
		datapoint.New("memory.used", dimensions, datapoint.NewIntValue(int64(memInfo.Used)), datapoint.Gauge, time.Time{}),
		datapoint.New("memory.available", dimensions, datapoint.NewIntValue(int64(memInfo.Available)), datapoint.Gauge, time.Time{}),
		datapoint.New("memory.total", dimensions, datapoint.NewIntValue(int64(memInfo.Total)), datapoint.Gauge, time.Time{}),
	}
}
