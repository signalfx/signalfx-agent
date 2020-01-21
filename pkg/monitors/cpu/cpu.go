package cpu

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

const cpuUtilName = "cpu.utilization"
const percoreMetricName = "cpu.utilization_per_core"

var errorUsedDiffLessThanZero = fmt.Errorf("usedDiff < 0")
var errorTotalDiffLessThanZero = fmt.Errorf("totalDiff < 0")

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
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
	logger          logrus.FieldLogger
}

func (m *Monitor) generatePerCoreDatapoints() []*datapoint.Datapoint {
	totals, err := times(true)
	if err != nil {
		if err == context.DeadlineExceeded {
			m.logger.WithField("debug", err).Debugf("unable to get per core cpu times will try again in the next reporting cycle")
		} else {
			m.logger.WithField("warning", err).Warningf("unable to get per core cpu times will try again in the next reporting cycle")
		}
	}

	out := make([]*datapoint.Datapoint, 0, len(totals))
	// for each core
	for i := range totals {
		core := totals[i]
		// get current times as totalUsed
		current := cpuTimeStatTototalUsed(&core)

		// calculate utilization
		if prev, ok := m.previousPerCore[core.CPU]; ok && prev != nil {
			utilization, err := getUtilization(prev, current)

			if err != nil {
				m.logger.WithError(err).Errorf("failed to calculate utilization for cpu core %s", core.CPU)
				continue
			}

			// add datapoint to be returned
			out = append(out,
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

	return out
}

func (m *Monitor) generateDatapoints() []*datapoint.Datapoint {
	total, err := times(false)
	if err != nil || len(total) == 0 {
		if err == context.DeadlineExceeded {
			m.logger.WithField("debug", err).Debugf("unable to get cpu times will try again in the next reporting cycle")
		} else {
			m.logger.WithError(err).Errorf("unable to get cpu times will try again in the next reporting cycle")
		}
		return nil
	}
	// get current times as totalUsed
	current := cpuTimeStatTototalUsed(&total[0])

	dps := make([]*datapoint.Datapoint, 0)
	// calculate utilization
	if m.previousTotal != nil {
		utilization, err := getUtilization(m.previousTotal, current)

		// append errors
		if err != nil {
			if err == errorTotalDiffLessThanZero || err == errorUsedDiffLessThanZero {
				m.logger.WithField("debug", err).Debugf("failed to calculate utilization for cpu")
			} else {
				m.logger.WithError(err).Errorf("failed to calculate utilization for cpu")
			}
			return nil
		}

		// add datapoint to be returned
		dps = append(dps, datapoint.New(
			cpuUtilName,
			map[string]string{"plugin": types.UtilizationMetricPluginName},
			datapoint.NewFloatValue(utilization),
			datapoint.Gauge,
			time.Time{},
		))
	}

	// store current as previous value for next time
	m.previousTotal = current

	return dps
}

func getUtilization(prev *totalUsed, current *totalUsed) (utilization float64, err error) {
	if prev.Total == 0 {
		err = fmt.Errorf("prev.Total == 0 will skip until previous Total is > 0")
		return
	}

	usedDiff := current.Used - prev.Used
	totalDiff := current.Total - prev.Total
	switch {
	case usedDiff < 0:
		err = errorUsedDiffLessThanZero
	case totalDiff < 0:
		err = errorTotalDiffLessThanZero
	case (usedDiff == 0 && totalDiff == 0) || totalDiff == 0:
		utilization = 0
	default:
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
		m.logger.WithField("debug", err).Debugf("unable to initialize cpu times will try again in the next reporting cycle")
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
		m.logger.WithField("debug", err).Debugf("unable to initialize per core cpu times will try again in the next reporting cycle")
	}
	m.previousPerCore = make(map[string]*totalUsed, len(totals))
	for i := range totals {
		m.previousPerCore[totals[i].CPU] = cpuTimeStatTototalUsed(&totals[i])
	}
}

// Configure is the main function of the monitor, it will report host metadata
// on a varied interval
func (m *Monitor) Configure(conf *Config) error {
	m.logger = logrus.WithFields(log.Fields{"monitorType": monitorType})
	if runtime.GOOS != "windows" {
		m.logger.Warningf("'%s' monitor is in beta on this platform.  For production environments please use 'collectd/%s'.", monitorType, monitorType)
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
		dps := m.generateDatapoints()
		// NOTE: If this monitor ever fails to complete in a reporting interval
		// maybe run this on a separate go routine
		perCoreDPs := m.generatePerCoreDatapoints()

		m.Output.SendDatapoints(append(dps, perCoreDPs...)...)
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
		t.GuestNice

	return &totalUsed{
		Total: total,
		Used:  total - t.Idle,
	}
}
