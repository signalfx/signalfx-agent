package perfhelper

import (
	"fmt"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/measurement"
)

// MetricMapper is used for mapping a measurement to a metric name, type, plugin/monitor name, and instance dim.
type MetricMapper struct {
	// The metric name to use
	Name string
	// The metric type to emit with
	Type datapoint.MetricType
	// The monitor name / plugin name
	Monitor string
	// The key to use for the instance dimension
	Instance string
}

// ProcessMeasurements processes an array of measurements and map of mappings emitting them to the supplied emitter functions
func ProcessMeasurements(measurements []*measurement.Measurement, mappings map[string]*MetricMapper, sendDatapoint func(dp *datapoint.Datapoint), defaultMonitorType string, defaultInstanceKey string) (errors []error) {
	for _, ms := range measurements {
		if len(ms.Fields) == 0 {
			errors = append(errors, fmt.Errorf("no fields on measurement '%s'", ms.Measurement))
			return
		}

		// retrieve the plugin instance
		var ok bool
		var instanceVal string
		if instanceVal, ok = ms.Tags["instance"]; !ok {
			errors = append(errors, fmt.Errorf("no instance tag defined in tags '%v' for measurement '%s'",
				ms.Tags, ms.Measurement))
			continue
		}

		for field, val := range ms.Fields {
			var metricType = datapoint.Gauge
			var instanceKey = defaultInstanceKey
			var monitorType = defaultMonitorType
			metricName := fmt.Sprintf("%s.%s", ms.Measurement, field)

			if mapping := mappings[metricName]; mapping != nil {
				// use the mapping if it exists
				if mapping.Name != "" {
					metricName = mapping.Name
				}
				if mapping.Type != metricType {
					metricType = mapping.Type
				}
				if mapping.Instance != "" {
					instanceKey = mapping.Instance
				}
				if mapping.Monitor != "" {
					monitorType = mapping.Monitor
				}
			}

			dimensions := map[string]string{"plugin": monitorType}
			if instanceKey != "" && instanceVal != "" {
				dimensions[instanceKey] = instanceVal
			}

			// parse metric value
			var metricVal datapoint.Value
			var err error
			if metricVal, err = datapoint.CastMetricValue(val); err != nil {
				errors = append(errors, err)
				continue
			}

			dp := datapoint.New(metricName, dimensions, metricVal, metricType, time.Time{})
			sendDatapoint(dp)
		}
	}
	return
}
