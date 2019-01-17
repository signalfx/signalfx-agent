// +build windows

package utilization

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/mem"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/accumulator"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/emitter/batchemitter"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/measurement"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/winperfcounters"
	"github.com/signalfx/signalfx-agent/internal/monitors/utilization/perfcounter"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

// emitMemoryUtilizaion collects and emits memory metrics via gopstuil.
// We do this because we can't get all of the memory metrics via perf counter
// on all supported win versions.
func (m *Monitor) emitMemoryUtilization() {
	// mem.VirtualMemory is a gopsutil function
	memInfo, _ := mem.VirtualMemory()
	dimensions := map[string]string{"plugin": monitorType}
	// perfcounter: ""; perfcounter reporter: "memory.available_mbytes"; collectd: "memory.free";
	m.Output.SendDatapoint(datapoint.New("memory.free", dimensions, datapoint.NewIntValue(int64(memInfo.Available)), datapoint.Gauge, time.Now()))
	// perfcounter: ""; perfcounter reporter: "signalfx.usedmemory"; collectd: "memory.used";
	m.Output.SendDatapoint(datapoint.New("memory.used", dimensions, datapoint.NewIntValue(int64(memInfo.Used)), datapoint.Gauge, time.Now()))
	// perfcounter: ""; perfcounter reporter: ""; collectd: "memory.utilization"
	util := (float64(memInfo.Used) / float64(memInfo.Total)) * 100
	m.Output.SendDatapoint(datapoint.New("memory.utilization", dimensions, datapoint.NewFloatValue(util), datapoint.Gauge, time.Now()))
}

func (m *Monitor) getPerfCounters() ([]winperfcounters.PerfCounterObj, func([]*measurement.Measurement)) {
	memory := perfcounter.Memory()
	processor := perfcounter.Processor()
	logicalDisk := perfcounter.LogicalDisk()
	physicalDisk := perfcounter.PhysicalDisk()
	networkIntf := perfcounter.NetworkInterface()
	pagefile := perfcounter.PageFile()
	processMeasurements := func(measurements []*measurement.Measurement) {
		for _, measurement := range measurements {
			var err []error
			switch measurement.Measurement {
			case logicalDisk.Measurement():
				err = logicalDisk.ProcessMeasurement(measurement, monitorType, m.Output.SendDatapoint)
			case physicalDisk.Measurement():
				err = physicalDisk.ProcessMeasurement(measurement, monitorType, m.Output.SendDatapoint)
			case processor.Measurement():
				err = processor.ProcessMeasurement(measurement, monitorType, m.Output.SendDatapoint)
			case networkIntf.Measurement():
				err = networkIntf.ProcessMeasurement(measurement, monitorType, m.Output.SendDatapoint)
			case memory.Measurement():
				err = memory.ProcessMeasurement(measurement, monitorType, m.Output.SendDatapoint)
			case pagefile.Measurement():
				logger.Debugf("%v", measurement)
				err = pagefile.ProcessMeasurement(measurement, monitorType, m.Output.SendDatapoint)
			default:
				logger.Errorf("utilization plugin collected unknown measurement %v", measurement)
			}
			// log errors
			for _, e := range err {
				logger.Error(e.Error())
			}
		}
	}
	return []winperfcounters.PerfCounterObj{
		memory.PerfCounterObj(),
		processor.PerfCounterObj(),
		logicalDisk.PerfCounterObj(),
		physicalDisk.PerfCounterObj(),
		networkIntf.PerfCounterObj(),
		pagefile.PerfCounterObj(),
	}, processMeasurements
}

// Configure the monitor and kick off metric syncing
func (m *Monitor) Configure(conf *Config) error {
	perfCounters, processMeasurements := m.getPerfCounters()
	perfcounterConf := &winperfcounters.Config{
		CountersRefreshInterval: conf.CountersRefreshInterval,
		PrintValid:              conf.PrintValid,
		Object:                  perfCounters,
	}

	plugin := winperfcounters.GetPlugin(perfcounterConf)

	// create batch emitter
	emitter := batchemitter.NewEmitter(m.Output, logger)

	// create the accumulator
	ac := accumulator.NewAccumulator(emitter)

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		if err := plugin.Gather(ac); err != nil {
			logger.Error(err)
		}

		// process measurements retrieved from perf counter monitor
		processMeasurements(emitter.Measurements)

		// reset batch emitter
		// NOTE: we can do this here because this emitter is on a single routine
		// if that changes, make sure you lock the mutex on the batch emitter
		emitter.Measurements = emitter.Measurements[:0]

		// memory utilization is collected from gopsutil instead of through perf counter
		// NOTE: if more helper functions are need to collect data we might want to put them on separate routines
		m.emitMemoryUtilization()
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
