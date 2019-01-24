package cpu

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/gopsutilhelper"
	log "github.com/sirupsen/logrus"
)

const monitorType = "cpu"
const cpuUtilName = "cpu.utilization"
const percoreMetricName = "cpu.utilization_per_core"

// setting cpu.Times to a package variable for testing purposes
var times = cpu.Times

// MONITOR(cpu):
// This monitor reports cpu and cpu utilization metrics.
//
//
// ```yaml
// monitors:
//  - type: cpu
// ```

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

type totalUsed struct {
	Total float64
	Used  float64
}

// TODO: make ProcFSPath a global config

// Config for this monitor
type Config struct {
	config.MonitorConfig `singleInstance:"true" acceptsEndpoints:"false"`
	// The path to the proc filesystem. Useful to override in containerized
	// environments.  (Does not apply to windows)
	ProcFSPath      string `yaml:"procFSPath" default:"/proc"`
	previousPerCore map[string]*totalUsed
	previousTotal   *totalUsed
}

// cpuTimeStatTototalUsed converts a cpu.TimesStat to a totalUsed with Total and Used values
func cpuTimeStatTototalUsed(t *cpu.TimesStat) *totalUsed {
	// add up all times if a value doesn't apply then the struct field
	// will be 0 and shouldn't affect anything
	total := t.User +
		t.System +
		t.Idle +
		t.Nice +
		t.Iowait +
		t.Irq +
		t.Softirq +
		t.Steal +
		t.Guest +
		t.GuestNice +
		t.Stolen

	return &totalUsed{
		Total: total,
		Used:  total - t.Idle,
	}
}

func (m *Monitor) emitPerCoreDatapoints(conf *Config) {
	totals, err := times(true)
	if err != nil {
		logger.WithError(err).Errorf("unable to get cpu times per core")
	}
	// for each core
	for _, core := range totals {
		// get current times as totalUsed
		current := cpuTimeStatTototalUsed(&core)

		// calculate utilization
		if prev, ok := conf.previousPerCore[core.CPU]; ok && prev != nil {
			utilization, err := getUtilization(prev, current)

			if err != nil {
				logger.WithError(err).Errorf("failed to calculate utilization for cpu core %s", core.CPU)
				continue
			}

			// add datapoint to be returned
			m.Output.SendDatapoint(
				datapoint.New(
					percoreMetricName,
					map[string]string{"plugin": types.UtilizationMetricPluginName, "core": core.CPU},
					datapoint.NewFloatValue(utilization),
					datapoint.Gauge,
					time.Time{},
				))
		}

		// store current as previous value for next time
		conf.previousPerCore[core.CPU] = current
	}
}

func (m *Monitor) emitDatapoints(conf *Config) {
	total, err := times(false)
	if err != nil {
		logger.WithError(err).Errorf("unable to get cpu times")
	}
	if len(total) > 0 {
		// get current times as totalUsed
		current := cpuTimeStatTototalUsed(&total[0])

		// calculate utilization
		if conf.previousTotal != nil {
			utilization, err := getUtilization(conf.previousTotal, current)

			// append errors
			if err != nil {
				logger.WithError(err).Errorf("failed to calculate utilization for cpu")
				return
			}

			// add datapoint to be returned
			m.Output.SendDatapoint(
				datapoint.New(
					cpuUtilName,
					map[string]string{"plugin": types.UtilizationMetricPluginName},
					datapoint.NewFloatValue(utilization),
					datapoint.Gauge,
					time.Time{},
				))
		}

		// store current as previous value for next time
		conf.previousTotal = current
	}

	return
}

func getUtilization(prev *totalUsed, current *totalUsed) (utilization float64, err error) {
	if prev.Total == 0 {
		err = fmt.Errorf("prev.Total == 0 will skip until previous Total is > 0")
	}

	usedDiff := current.Used - prev.Used
	totalDiff := current.Total - prev.Total
	if usedDiff < 0 || totalDiff < 0 {
		err = fmt.Errorf("usedDiff (%v) or totalDiff (%v) are < 0", usedDiff, totalDiff)
	} else if (usedDiff == 0 && totalDiff == 0) || totalDiff == 0 {
		utilization = 0
	} else {
		// calculate utilization
		utilization = usedDiff / totalDiff * 100
		if utilization < 0 {
			err = fmt.Errorf("percent %v < 0 total: %v used: %v", utilization, usedDiff, totalDiff)
		}
		if utilization > 100 {
			err = fmt.Errorf("percent %v > 100 total: %v used: %v ", utilization, usedDiff, totalDiff)
		}
	}

	return
}

func initializeCPUTimes(conf *Config) {
	// initialize previous values
	var total []cpu.TimesStat
	var err error
	if total, err = times(false); err != nil {
		logger.WithError(err).Errorf("failed to initialize cpu times")
	}
	if len(total) > 0 {
		conf.previousTotal = cpuTimeStatTototalUsed(&total[0])
	}
}

func initializePerCoreCPUTimes(conf *Config) {
	// initialize per core cpu times
	var totals []cpu.TimesStat
	var err error
	if totals, err = times(true); err != nil {
		logger.WithError(err).Errorf("failed to initialize per core cpu times")
	}
	conf.previousPerCore = make(map[string]*totalUsed, len(totals))
	for _, core := range totals {
		conf.previousPerCore[core.CPU] = cpuTimeStatTototalUsed(&core)
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

	// initialize cpu times and per core cpu times so that we don't have to wait an entire reporting interval to report utilization
	initializeCPUTimes(conf)
	initializePerCoreCPUTimes(conf)

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		m.emitDatapoints(conf)
		// NOTE: If this monitor ever fails to complete in a reporting interval
		// maybe run this on a separate go routine
		m.emitPerCoreDatapoints(conf)
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
