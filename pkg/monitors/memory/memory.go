package memory

import (
	"context"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/mem"
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

const windowsOS = "windows"

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `singleInstance:"true" acceptsEndpoints:"false"`
}

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel func()
	logger logrus.FieldLogger
}

func (m *Monitor) makeDatapointsWindows(memInfo *mem.VirtualMemoryStat, dimensions map[string]string) []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		datapoint.New("memory.available", dimensions, datapoint.NewIntValue(int64(memInfo.Available)), datapoint.Gauge, time.Time{}),
	}
}

func (m *Monitor) makeDatapointsNotWindows(memInfo *mem.VirtualMemoryStat, dimensions map[string]string) []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		datapoint.New("memory.free", dimensions, datapoint.NewIntValue(int64(memInfo.Free)), datapoint.Gauge, time.Time{}),
	}
}

func (m *Monitor) makeDatapointsDarwin(memInfo *mem.VirtualMemoryStat, dimensions map[string]string) []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		datapoint.New("memory.active", dimensions, datapoint.NewIntValue(int64(memInfo.Active)), datapoint.Gauge, time.Time{}),
		datapoint.New("memory.inactive", dimensions, datapoint.NewIntValue(int64(memInfo.Inactive)), datapoint.Gauge, time.Time{}),
		datapoint.New("memory.wired", dimensions, datapoint.NewIntValue(int64(memInfo.Wired)), datapoint.Gauge, time.Time{}),
	}
}

func (m *Monitor) makeDatapointsLinux(memInfo *mem.VirtualMemoryStat, dimensions map[string]string) []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		datapoint.New("memory.buffered", dimensions, datapoint.NewIntValue(int64(memInfo.Buffers)), datapoint.Gauge, time.Time{}),
		// for some reason gopsutil decided to add slab_reclaimable to cached which collectd does not
		datapoint.New("memory.cached", dimensions, datapoint.NewIntValue(int64(memInfo.Cached-memInfo.SReclaimable)), datapoint.Gauge, time.Time{}),
		datapoint.New("memory.slab_recl", dimensions, datapoint.NewIntValue(int64(memInfo.SReclaimable)), datapoint.Gauge, time.Time{}),
		datapoint.New("memory.slab_unrecl", dimensions, datapoint.NewIntValue(int64(memInfo.Slab-memInfo.SReclaimable)), datapoint.Gauge, time.Time{}),
	}
}

// EmitDatapoints emits a set of memory datapoints
func (m *Monitor) emitDatapoints() {
	// mem.VirtualMemory is a gopsutil function
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		if err == context.DeadlineExceeded {
			m.logger.WithField("debug", err).Debugf("unable to collect memory time info")
		} else {
			m.logger.WithError(err).Errorf("unable to collect memory time info")
		}
		return
	}

	dimensions := map[string]string{"plugin": monitorType}

	// all platforms
	dps := []*datapoint.Datapoint{datapoint.New("memory.utilization", map[string]string{"plugin": types.UtilizationMetricPluginName}, datapoint.NewFloatValue(memInfo.UsedPercent), datapoint.Gauge, time.Time{}),
		datapoint.New("memory.used", dimensions, datapoint.NewIntValue(int64(memInfo.Used)), datapoint.Gauge, time.Time{}),
	}

	// windows only
	if runtime.GOOS == windowsOS {
		dps = append(dps, m.makeDatapointsWindows(memInfo, dimensions)...)
	}

	// linux + darwin only
	if runtime.GOOS != windowsOS {
		dps = append(dps, m.makeDatapointsNotWindows(memInfo, dimensions)...)
	}

	// darwin only
	if runtime.GOOS == "darwin" {
		dps = append(dps, m.makeDatapointsDarwin(memInfo, dimensions)...)
	}

	// linux only
	if runtime.GOOS == "linux" {
		dps = append(dps, m.makeDatapointsLinux(memInfo, dimensions)...)
	}

	m.Output.SendDatapoints(dps...)
}

// Configure is the main function of the monitor, it will report host metadata
// on a varied interval
func (m *Monitor) Configure(conf *Config) error {
	m.logger = logrus.WithFields(log.Fields{"monitorType": monitorType})
	if runtime.GOOS != windowsOS {
		m.logger.Warningf("'%s' monitor is in beta on this platform.  For production environments please use 'collectd/%s'.", monitorType, monitorType)
	}

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

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
