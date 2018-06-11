package emitter

import (
	"time"

	"github.com/signalfx/golib/datapoint"
)

// Emitter interface to telegraf accumulator for processing metrics from
// telegraf
type Emitter interface {
	// Add is a function used by the telegraf accumulator to emit events
	// through the agent.  Pleaes note that if the emitter is a BatchEmitter
	// you will have to invoke the Send() function to send the batch of
	// datapoints and events collected by the Emit function
	Add(measurement string, fields map[string]interface{}, tags map[string]string, metricType datapoint.MetricType, originalMetricType string, t ...time.Time)
	// AddTag adds a key/value pair to all measurement tags.  If a key conflicts
	// the key value pair in AddTag will override the original key on the
	// measurement
	AddTag(key string, val string)
	// AddTags adds a map of key value pairs to all measurement tags.  If a key
	// conflicts the key value pair in AddTags will override the original key on
	// the measurement.
	AddTags(tags map[string]string)
	// IncludeEvent a thread safe function for registering an event name to
	// include during emission. We disable all events by default because
	// Telegraf has some junk events.
	IncludeEvent(name string)
	// IncludeEvents is a thread safe function for registering a list of event
	// names to include during emission. We disable all events by default
	// because Telegraf has some junk events.
	IncludeEvents(names []string)
	// ExcludeDatum adds a name to the list of metrics and events to
	// exclude
	ExcludeDatum(name string)
	// ExcludeData adds a list of names the list of metrics and events
	// to exclude
	ExcludeData(names []string)
	// OmitTag adds a tag to the list of tags to remove from measurements
	OmitTag(tag string)
	// OmitTags adds a list of tags the list of tags to remove from measurements
	OmitTags(tags []string)
	// AddError handles errors added to the accumulator by telegraf plugins
	// the default behavior is to log the error
	AddError(err error)
}
