package memory

import (
	"context"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/mem"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/gopsutilhelper"
	log "github.com/sirupsen/logrus"
)

const monitorType = "memory"

// setting mem.VirtualMemory to a package variable for testing purposes
var virtualMemory = mem.VirtualMemory

// MONITOR(memory):
// This monitor reports memory and memory utilization metrics.
//
//
// ```yaml
// monitors:
//  - type: memory
// ```

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

// TODO: make ProcFSPath a global config

// Config for this monitor
type Config struct {
	config.MonitorConfig `singleInstance:"true" acceptsEndpoints:"false"`
	// The path to the proc filesystem. Useful to override in containerized
	// environments.  (Does not apply to windows)
	ProcFSPath string `yaml:"procFSPath" default:"/proc"`
}

func (m *Monitor) processDatapointsWindows(memInfo *mem.VirtualMemoryStat, dimensions map[string]string) {
	m.Output.SendDatapoint(datapoint.New("memory.available", dimensions, datapoint.NewIntValue(int64(memInfo.Available)), datapoint.Gauge, time.Time{}))
}

func (m *Monitor) processDatapointsNotWindows(memInfo *mem.VirtualMemoryStat, dimensions map[string]string) {
	m.Output.SendDatapoint(datapoint.New("memory.free", dimensions, datapoint.NewIntValue(int64(memInfo.Free)), datapoint.Gauge, time.Time{}))
}

func (m *Monitor) processDatapointsDarwin(memInfo *mem.VirtualMemoryStat, dimensions map[string]string) {
	m.Output.SendDatapoint(datapoint.New("memory.active", dimensions, datapoint.NewIntValue(int64(memInfo.Active)), datapoint.Gauge, time.Time{}))
	m.Output.SendDatapoint(datapoint.New("memory.inactive", dimensions, datapoint.NewIntValue(int64(memInfo.Inactive)), datapoint.Gauge, time.Time{}))
	m.Output.SendDatapoint(datapoint.New("memory.wired", dimensions, datapoint.NewIntValue(int64(memInfo.Wired)), datapoint.Gauge, time.Time{}))
}

func (m *Monitor) processDatapointsLinux(memInfo *mem.VirtualMemoryStat, dimensions map[string]string) {
	m.Output.SendDatapoint(datapoint.New("memory.buffered", dimensions, datapoint.NewIntValue(int64(memInfo.Buffers)), datapoint.Gauge, time.Time{}))
	// for some reason gopsutil decided to add slab_reclaimable to cached which collectd does not
	m.Output.SendDatapoint(datapoint.New("memory.cached", dimensions, datapoint.NewIntValue(int64(memInfo.Cached-memInfo.SReclaimable)), datapoint.Gauge, time.Time{}))
	m.Output.SendDatapoint(datapoint.New("memory.slab_recl", dimensions, datapoint.NewIntValue(int64(memInfo.SReclaimable)), datapoint.Gauge, time.Time{}))
	m.Output.SendDatapoint(datapoint.New("memory.slab_unrecl", dimensions, datapoint.NewIntValue(int64(memInfo.Slab-memInfo.SReclaimable)), datapoint.Gauge, time.Time{}))
}

// EmitDatapoints emits a set of memory datapoints
func (m *Monitor) emitDatapoints() {
	// mem.VirtualMemory is a gopsutil function
	memInfo, err := virtualMemory()
	if err != nil {
		logger.WithError(err).Errorf("unable to collect memory time info")
		return
	}

	dimensions := map[string]string{"plugin": monitorType}

	// all platforms
	m.Output.SendDatapoint(datapoint.New("memory.utilization", map[string]string{"plugin": constants.UtilizationMetricPluginName}, datapoint.NewFloatValue(memInfo.UsedPercent), datapoint.Gauge, time.Time{}))
	m.Output.SendDatapoint(datapoint.New("memory.used", dimensions, datapoint.NewIntValue(int64(memInfo.Used)), datapoint.Gauge, time.Time{}))

	// windows only
	if runtime.GOOS == "windows" {
		m.processDatapointsWindows(memInfo, dimensions)
	}

	// linux + darwin only
	if runtime.GOOS != "windows" {
		m.processDatapointsNotWindows(memInfo, dimensions)
	}

	// darwin only
	if runtime.GOOS == "darwin" {
		m.processDatapointsDarwin(memInfo, dimensions)
	}

	// linux only
	if runtime.GOOS == "linux" {
		m.processDatapointsLinux(memInfo, dimensions)
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

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel func()
}
