package accumulator

import (
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/emitter"
)

// Accumulator is an interface used to accumulate telegraf measurements from
// Telegraf plugins.
type Accumulator struct {
	emit emitter.Emitter
}

// AddFields receives a measurement with tags and a time stamp to the accumulator.
// Measurements are passed to the Accumulator's Emitter.
func (ac *Accumulator) AddFields(measurement string, fields map[string]interface{},
	tags map[string]string, t ...time.Time) {
	ac.emit.Add(measurement, fields, tags, datapoint.Gauge, "untyped", t...)
}

// AddGauge receives a measurement as a "Gauge" with tags and a time stamp to
// the accumulator. Measurements are passed to the Accumulator's Emitter.
func (ac *Accumulator) AddGauge(measurement string, fields map[string]interface{},
	tags map[string]string, t ...time.Time) {
	ac.emit.Add(measurement, fields, tags, datapoint.Gauge, "", t...)
}

// AddCounter receives a measurement as a "Counter" with tags and a time stamp
// to the accumulator. Measurements are passed to the Accumulator's Emitter.
func (ac *Accumulator) AddCounter(measurement string, fields map[string]interface{},
	tags map[string]string, t ...time.Time) {
	ac.emit.Add(measurement, fields, tags, datapoint.Counter, "", t...)
}

// AddSummary receives a measurement as a "Counter" with tags and a time stamp
// to the accumulator. Measurements are passed to the Accumulator's Emitter.
func (ac *Accumulator) AddSummary(measurement string, fields map[string]interface{},
	tags map[string]string, t ...time.Time) {
	ac.emit.Add(measurement, fields, tags, datapoint.Gauge, "summary", t...)
}

// AddHistogram receives a measurement as a "Counter" with tags and a time stamp
// to the accumulator. Measurements are passed to the Accumulator's Emitter.
func (ac *Accumulator) AddHistogram(measurement string, fields map[string]interface{},
	tags map[string]string, t ...time.Time) {
	ac.emit.Add(measurement, fields, tags, datapoint.Gauge, "histogram", t...)
}

// SetPrecision - SignalFx does not implement this
func (ac *Accumulator) SetPrecision(precision, interval time.Duration) {
}

// AddError - log an error returned by the plugin
func (ac *Accumulator) AddError(err error) {
	ac.emit.AddError(err)
}

// NewAccumulator returns a pointer to an Accumulator
func NewAccumulator(e emitter.Emitter) *Accumulator {
	return &Accumulator{
		emit: e,
	}
}
