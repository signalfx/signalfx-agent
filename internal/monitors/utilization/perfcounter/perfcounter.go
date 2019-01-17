package perfcounter

import (
	"fmt"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/pointer"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/measurement"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/winperfcounters"
)

// PerfCounter is a performance counter object for the utilization plugin
type PerfCounter interface {
	// Measurement returns the measurement string for the performance counter
	Measurement() string
	// PerfCounterObj returns the windows performance counter object for the performance counter monitor
	PerfCounterObj() winperfcounters.PerfCounterObj
	// ProcessMeasurement takes a measurement and emits it through the provided sendDatapoint function
	ProcessMeasurement(ms *measurement.Measurement, monitorType string, sendDatapoint func(dp *datapoint.Datapoint)) []error
}

// BasePerfCounter is a base performance counter struct
type BasePerfCounter struct {
	// objectName
	objectName string
	// measurement
	measurement string
	// includeTotal
	includeTotal bool
	// counters are the counters to collect
	counters []string
	// instances are the instances to collect
	instances []string
	// instanceDimensionKey
	instanceDimensionKey string
	// metricType
	metricType datapoint.MetricType
	// metricNameMapping
	metricNameMapping map[string]string
	// warnOnMissing
	warnOnMissing bool
}

// Measurement returns the measurement string for the performance counter
func (p *BasePerfCounter) Measurement() string {
	return p.measurement
}

// PerfCounterObj returns the performance counter object for the performance counter monitor
func (p *BasePerfCounter) PerfCounterObj() winperfcounters.PerfCounterObj {
	return winperfcounters.PerfCounterObj{
		ObjectName:    p.objectName,
		Counters:      p.counters,
		Instances:     p.instances,
		Measurement:   p.measurement,
		IncludeTotal:  p.includeTotal,
		WarnOnMissing: p.warnOnMissing,
	}
}

// ProcessMeasurement processes a given measurement and returns metrics
func (p *BasePerfCounter) ProcessMeasurement(ms *measurement.Measurement, monitorType string, sendDatapoint func(dp *datapoint.Datapoint)) (errors []error) {
	if len(ms.Fields) == 0 {
		errors = append(errors, fmt.Errorf("no fields on measurement '%s'", ms.Measurement))
		return
	}

	dimensions := map[string]string{"plugin": monitorType}
	if p.instanceDimensionKey != "" {
		// retrieve the plugin instance
		var ok bool
		var instance string
		if instance, ok = ms.Tags["instance"]; !ok {
			errors = append(errors, fmt.Errorf("no instance tag defined in tags '%v' for measurement '%s'",
				ms.Tags, ms.Measurement))
			return
		}
		// dimensions for metrics
		dimensions[p.instanceDimensionKey] = instance
	}

	for field, val := range ms.Fields {
		// set metric name
		metricName := p.metricNameMapping[field]
		if metricName == "" {
			errors = append(errors, fmt.Errorf("unable to map field '%s' to a metricname while parsing measurement '%s'",
				field, ms.Measurement))
			continue
		}

		// parse metric value
		var metricVal datapoint.Value
		var err error
		if metricVal, err = datapoint.CastMetricValue(val); err != nil {
			errors = append(errors, err)
			continue
		}

		dp := datapoint.New(metricName, dimensions, metricVal, p.metricType, time.Time{})
		sendDatapoint(dp)
	}
	return
}

// LogicalDisk Metrics
// perfcounter: "% Free Space"; perfcounter reporter: "logicaldisk.pct_free_space"; collectd: "disk.utilization";
const diskUtilMetric = "disk.utilization"

// perfcounter: "Free Megabytes"; perfcounter reporter: "logicaldisk.free_megabytes"; collectd: "df_complex.free";
const diskFreeMetric = "df_complex.free"

// perfcounter: "N/A"; perfcounter reporter: "N/A"; collectd: "df_complex.used";
const diskUsedMetric = "df_complex.used"

const megabytesToBytes float64 = 1048576

// LogicalDiskImpl is a PerfCounter for LogicalDisk measurements
type LogicalDiskImpl struct{ *BasePerfCounter }

// ProcessMeasurement processes a given measurement and returns logical disk
func (l *LogicalDiskImpl) ProcessMeasurement(ms *measurement.Measurement, monitorType string, sendDatapoint func(dp *datapoint.Datapoint)) (errors []error) {
	if len(ms.Fields) == 0 {
		errors = append(errors, fmt.Errorf("no fields on logical disk measurement '%s'", ms.Measurement))
		return
	}

	var dimensions map[string]string

	// check for a plugin instance
	var ok bool
	var instance string
	if instance, ok = ms.Tags["instance"]; !ok {
		errors = append(errors, fmt.Errorf("no instance tag defined in tags '%v' for measurement '%s'", ms.Tags, ms.Measurement))
		return
	}

	// set the dimensions map
	dimensions = map[string]string{"plugin": monitorType, l.instanceDimensionKey: instance}

	var utilization *float64
	if val, ok := ms.Fields["Percent_Free_Space"]; ok {
		if v, ok := val.(float32); ok {
			utilization = pointer.Float64(float64(100.0 - v))
			sendDatapoint(datapoint.New(diskUtilMetric, dimensions, datapoint.NewFloatValue(*utilization), l.metricType, time.Time{}))
		} else {
			errors = append(errors, fmt.Errorf("error parsing value '%v' for 'Percent_Free_Space' field in logical disk measurement '%s'", val, ms.Measurement))
			return
		}
	} else {
		errors = append(errors, fmt.Errorf("No 'Percent_Free_Space' field on logical disk measurement '%s'", ms.Measurement))
	}

	var used float64
	var free float64
	if val, ok := ms.Fields["Free_Megabytes"]; ok {
		if v, ok := val.(float32); ok {
			free = (float64(v) * megabytesToBytes)
			sendDatapoint(datapoint.New(diskFreeMetric, dimensions, datapoint.NewFloatValue(free), l.metricType, time.Time{}))

			if utilization != nil {
				used = ((free * 100) / (100 - *utilization)) - free
				sendDatapoint(datapoint.New(diskUsedMetric, dimensions, datapoint.NewFloatValue(used), l.metricType, time.Time{}))
			}
		} else {
			errors = append(errors, fmt.Errorf("error parsing value '%v' for 'Free_Megabytes' field in logical disk measurement '%s'", val, ms.Measurement))
		}
	}
	return
}

// ProcessorImpl is a PerfCounter for Processor measurements
type ProcessorImpl struct{ *BasePerfCounter }

// ProcessMeasurement processes a given measurement and returns cpu metrics
func (p *ProcessorImpl) ProcessMeasurement(ms *measurement.Measurement, monitorType string, sendDatapoint func(dp *datapoint.Datapoint)) (errors []error) {
	if len(ms.Fields) == 0 {
		errors = append(errors, fmt.Errorf("no fields on processor measurement '%s'", ms.Measurement))
		return
	}
	var metricName string
	var dimensions map[string]string

	// handle cpu utilization per core if instance isn't _Total
	if instance, ok := ms.Tags["instance"]; !ok {
		errors = append(errors, fmt.Errorf("no instance tag defined in tags '%v' on measurement '%s'",
			ms.Tags, ms.Measurement))
		return
	} else if instance == "_Total" {
		// perfcounter: "Processor"; perfcounter reporter: "processor.pct_processor_time"; collectd: "cpu.utilization";
		metricName = "cpu.utilization"
		dimensions = map[string]string{"plugin": monitorType}
	} else {
		// perfcounter: "Processor"; perfcounter reporter: "processor.pct_processor_time"; collectd: "cpu.utilization_per_core";
		metricName = "cpu.utilization_per_core"
		dimensions = map[string]string{"plugin": monitorType, "core": instance}
	}

	// parse metric value
	var metricVal datapoint.Value
	var err error
	if val, ok := ms.Fields["Percent_Processor_Time"]; ok {
		if metricVal, err = datapoint.CastMetricValue(val); err != nil {
			errors = append(errors, err)
			return
		}
		// create datapoint
		dp := datapoint.New(metricName, dimensions, metricVal, p.metricType, time.Time{})
		sendDatapoint(dp)
	}

	return
}

// LogicalDisk performance counter
func LogicalDisk() PerfCounter {
	return &LogicalDiskImpl{
		&BasePerfCounter{
			measurement:          "win_logical_disk",
			includeTotal:         true,
			warnOnMissing:        true,
			objectName:           "LogicalDisk",
			instanceDimensionKey: "plugin_instance",
			metricType:           datapoint.Gauge,
			instances:            []string{"*"},
			counters:             []string{"% Free Space", "Free Megabytes"},
		},
	}
}

// NetworkInterface performance counter
func NetworkInterface() PerfCounter {
	return &BasePerfCounter{
		measurement:          "win_network_interface",
		includeTotal:         true,
		warnOnMissing:        true,
		objectName:           "Network Interface",
		instanceDimensionKey: "interface",
		metricType:           datapoint.Gauge,
		instances:            []string{"*"},
		counters:             []string{"Bytes Received/sec", "Bytes Sent/sec", "Packets Received Errors", "Packets Outbound Errors"},
		metricNameMapping: map[string]string{
			// NetworkInterface - the original collectd metrics are cumulative counters, but these gauge the current rate
			"Bytes_Received_persec":   "network_interface.bytes_received.per_second",  // perfcounter: "Bytes Received/sec"; perfcounter reporter: ""; collectd: "if_octets.rx"
			"Bytes_Sent_persec":       "network_interface.bytes_sent.per_second",      // perfcounter: "Bytes Received/sec"; perfcounter reporter: ""; collectd: "if_octets.tx"
			"Packets_Received_Errors": "network_interface.errors_received.per_second", // perfcounter: ""; perfcounter reporter: ""; collectd: "if_errors.rx"
			"Packets_Outbound_Errors": "network_interface.errors_sent.per_second",     // perfcounter: ""; perfcounter reporter: ""; collectd: "if_errors.tx"
		},
	}
}

// PhysicalDisk performance counter
func PhysicalDisk() PerfCounter {
	return &BasePerfCounter{
		measurement:          "win_physical_disk",
		includeTotal:         true,
		warnOnMissing:        true,
		objectName:           "PhysicalDisk",
		instanceDimensionKey: "plugin_instance",
		metricType:           datapoint.Gauge,
		counters:             []string{"Disk Reads/sec", "Disk Writes/sec"},
		instances:            []string{"_Total"},
		metricNameMapping: map[string]string{
			// perfcounter: "Disk Reads/sec"; perfcounter reporter: ""; collectd: "disk_ops.read"
			"Disk_Reads_persec": "disk.read.per_second",
			// perfcounter: "Disk Writes/sec"; perfcounter reporter: ""; collectd: "disk_ops.write"
			"Disk_Writes_persec": "disk.write.per_second",
		},
	}
}

// PageFile performance counter
func PageFile() PerfCounter {
	return &BasePerfCounter{
		measurement:          "win_paging_file",
		includeTotal:         true,
		warnOnMissing:        true,
		objectName:           "Paging File",
		instanceDimensionKey: "instance",
		metricType:           datapoint.Gauge,
		counters:             []string{"% Usage"},
		instances:            []string{"*"},
		metricNameMapping: map[string]string{
			"Percent_Usage": "paging_file.pct_usage", // perfcounter: "% Usage"; perfcounter reporter: "paging_file.pct_usage"; collectd: "-";
		},
	}
}

// Memory performance counter
func Memory() PerfCounter {
	return &BasePerfCounter{
		measurement:          "win_memory",
		includeTotal:         true,
		warnOnMissing:        true,
		objectName:           "Memory",
		instanceDimensionKey: "", // no instance
		metricType:           datapoint.Gauge,
		counters:             []string{"Pages Input/sec", "Pages Output/sec", "Pages/sec"},
		instances:            []string{"------"},
		metricNameMapping: map[string]string{
			// Memory - the original collectd metrics are cumulative counters, but these gauge the current rate
			"Pages_Input_persec":  "vmpage.swapped_in.per_second",  // perfcounter: "Pages Input/sec"; perfcounter reporter: "memory.pages_input_sec"; collectd: "vmpage_io.swap.in";
			"Pages_Output_persec": "vmpage.swapped_out.per_second", // perfcounter: "Pages Input/sec"; perfcounter reporter: "-"; collectd: "vmpage_io.swap.out";
			"Pages_persec":        "vmpage.swapped.per_second",     // perfcounter: "Pages/sec"; perfcounter reporter: ""; collectd: "-";
		},
	}
}

// Processor performance counter
func Processor() PerfCounter {
	return &ProcessorImpl{
		&BasePerfCounter{
			measurement:   "win_cpu",
			includeTotal:  true,
			warnOnMissing: true,
			objectName:    "Processor",
			metricType:    datapoint.Gauge,
			counters:      []string{"% Processor Time"},
			instances:     []string{"*"},
		},
	}
}
