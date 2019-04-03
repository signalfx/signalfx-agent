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
	log "github.com/sirupsen/logrus"
)

const monitorType = "cpu"
const cpuUtilName = "cpu.utilization"
const percoreMetricName = "cpu.utilization_per_core"

var errorUsedDiffLessThanZero = fmt.Errorf("usedDiff < 0")
var errorTotalDiffLessThanZero = fmt.Errorf("totalDiff < 0")
var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `singleInstance:"true" acceptsEndpoints:"false"`
}

type totalUsed struct {
	Total float64
	Used  float64
}

// Monitor for Utilization
type Monitor struct {
	Output          types.Output
	cancel          func()
	conf            *Config
	previousPerCore map[string]*totalUsed
	previousTotal   *totalUsed
}

func (m *Monitor) emitPerCoreDatapoints() {
	totals, err := times(true)
	if err != nil {
		if err == context.DeadlineExceeded {
			logger.WithField("debug", err).Debugf("unable to get per core cpu times will try again in the next reporting cycle")
		} else {
			logger.WithField("warning", err).Warningf("unable to get per core cpu times will try again in the next reporting cycle")
		}
	}
	// for each core
	for _, core := range totals {
		// get current times as totalUsed
		current := cpuTimeStatTototalUsed(&core)

		// calculate utilization
		if prev, ok := m.previousPerCore[core.CPU]; ok && prev != nil {
			utilization, err := getUtilization(prev, current)

			if err != nil {
				logger.WithError(err).Errorf("failed to calculate utilization for cpu core %s", core.CPU)
				continue
			}

			// add datapoint to be returned
			m.Output.SendDatapoint(
				datapoint.New(
					percoreMetricName,
					map[string]string{"plugin": types.UtilizationMetricPluginName, "plugin_instance": core.CPU, "core": core.CPU},
					datapoint.NewFloatValue(utilization),
					datapoint.Gauge,
					time.Time{},
				))
		}

		// store current as previous value for next time
		m.previousPerCore[core.CPU] = current
	}
}

func (m *Monitor) emitDatapoints() {
	total, err := times(false)
	if err != nil || len(total) == 0 {
		if err == context.DeadlineExceeded {
			logger.WithField("debug", err).Debugf("unable to get cpu times will try again in the next reporting cycle")
		} else {
			logger.WithError(err).Errorf("unable to get cpu times will try again in the next reporting cycle")
		}
		return
	}
	// get current times as totalUsed
	current := cpuTimeStatTototalUsed(&total[0])

	// calculate utilization
	if m.previousTotal != nil {
		utilization, err := getUtilization(m.previousTotal, current)

		// append errors
		if err != nil {
			if err == errorTotalDiffLessThanZero || err == errorUsedDiffLessThanZero {
				logger.WithField("debug", err).Debugf("failed to calculate utilization for cpu")
			} else {
				logger.WithError(err).Errorf("failed to calculate utilization for cpu")
			}
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
	m.previousTotal = current
}

func getUtilization(prev *totalUsed, current *totalUsed) (utilization float64, err error) {
	if prev.Total == 0 {
		err = fmt.Errorf("prev.Total == 0 will skip until previous Total is > 0")
		return
	}

	usedDiff := current.Used - prev.Used
	totalDiff := current.Total - prev.Total
	if usedDiff < 0 {
		err = errorUsedDiffLessThanZero
	} else if totalDiff < 0 {
		err = errorTotalDiffLessThanZero
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

func (m *Monitor) initializeCPUTimes() {
	// initialize previous values
	var total []cpu.TimesStat
	var err error
	if total, err = times(false); err != nil {
		logger.WithField("debug", err).Debugf("unable to initialize cpu times will try again in the next reporting cycle")
	}
	if len(total) > 0 {
		m.previousTotal = cpuTimeStatTototalUsed(&total[0])
	}
}

func (m *Monitor) initializePerCoreCPUTimes() {
	// initialize per core cpu times
	var totals []cpu.TimesStat
	var err error
	if totals, err = times(true); err != nil {
		logger.WithField("debug", err).Debugf("unable to initialize per core cpu times will try again in the next reporting cycle")
	}
	m.previousPerCore = make(map[string]*totalUsed, len(totals))
	for _, core := range totals {
		m.previousPerCore[core.CPU] = cpuTimeStatTototalUsed(&core)
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

	// save config to monitor for convenience
	m.conf = conf

	// initialize cpu times and per core cpu times so that we don't have to wait an entire reporting interval to report utilization
	m.initializeCPUTimes()
	m.initializePerCoreCPUTimes()

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		m.emitDatapoints()
		// NOTE: If this monitor ever fails to complete in a reporting interval
		// maybe run this on a separate go routine
		m.emitPerCoreDatapoints()
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
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
