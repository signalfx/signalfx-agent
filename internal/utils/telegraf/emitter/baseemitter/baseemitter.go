package baseemitter

import (
	"sync"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/telegraf/plugins/outputs/signalfx/parse"
	log "github.com/sirupsen/logrus"
)

// GetTime returns the first timestamp from an array of timestamps
func GetTime(t ...time.Time) time.Time {
	if len(t) > 0 {
		return t[0]
	}
	return time.Now()
}

// BaseEmitter immediately converts a telegraf measurement into datapoints and
// sends them through Output
type BaseEmitter struct {
	lock   *sync.Mutex
	Output types.Output
	Logger *log.Entry
	// tagFitler is a tags that should be removed from measurements before being
	// processed
	tagFilter map[string]bool
	// Telegraf has some junk events so we exclude all events by default
	// and can enable them as needed by using IncludeEvent(string) or
	// IncludeEvents([]string).
	// You should look up included metrics using Included(string)bool.
	included map[string]bool
	// excluded metrics and events that should not be emitted.
	// You can add metrics and events to exclude by name using
	// ExcludeDatum(string) and ExcludeData(string).  You should look up
	// excluded events and metrics using Excluded(string)bool
	excluded map[string]bool
}

// IncludeEvent a thread safe function for registering an event name to include
// during emission. We disable all events by default because Telegraf has some
// junk events.
func (B *BaseEmitter) IncludeEvent(name string) {
	B.lock.Lock()
	B.included[name] = true
	B.lock.Unlock()
}

// IncludeEvents a thread safe function for registering a list of event names to
// include during emission. We disable all events by default because Telegraf
// has some junk events.
func (B *BaseEmitter) IncludeEvents(names []string) {
	for _, name := range names {
		B.IncludeEvent(name)
	}
}

// Included - A thread safe function for checking if events should be included
// during emission.  We disable all events by default because Telegraf has some
// junk events.
func (B *BaseEmitter) Included(name string) (included bool) {
	B.lock.Lock()
	included = B.included[name]
	B.lock.Unlock()
	return
}

// ExcludeDatum adds a name to the list of metrics and events to
// exclude
func (B *BaseEmitter) ExcludeDatum(name string) {
	B.lock.Lock()
	B.excluded[name] = true
	B.lock.Unlock()
}

// ExcludeData adds a list of names the list of metrics and events
// to exclude
func (B *BaseEmitter) ExcludeData(names []string) {
	for _, name := range names {
		B.ExcludeDatum(name)
	}
}

// Excluded - A thread safe function for checking if events or metrics should be
// excluded from emission
func (B *BaseEmitter) Excluded(name string) (excluded bool) {
	B.lock.Lock()
	excluded = B.excluded[name]
	B.lock.Unlock()
	return
}

// FilterTags - filter function for util.CloneAndFilterStringMapWithFunc()
// that will exclude any tags that
func (B *BaseEmitter) FilterTags(key string, value string) (include bool) {
	B.lock.Lock()
	include = B.tagFilter[key]
	B.lock.Unlock()
	return
}

// Add parses measurements from telegraf and emits them through Output
func (B *BaseEmitter) Add(measurement string, fields map[string]interface{},
	tags map[string]string, metricType datapoint.MetricType,
	originalMetricType string, t ...time.Time) {
	for field, val := range fields {
		// telegraf doc says that tags are owned by the calling plugin and they
		// shouldn't be mutated.  So we copy the tags map
		var metricDims = utils.CloneAndFilterStringMapWithFunc(tags, B.FilterTags)

		// Generate the metric name
		var metricName, isSFX = parse.GetMetricName(measurement, field, metricDims)

		// Check if the metric is explicitly excluded
		if B.Excluded(metricName) {
			B.Logger.Debugf("excluding the following metric: %s", metricName)
			continue
		}

		// If eligible, move the dimension "property" to properties
		metricProps, propErr := parse.ExtractProperty(metricName, metricDims)
		if propErr != nil {
			B.Logger.Error(propErr)
			continue
		}

		// Add common dimensions
		metricDims["telegraf_type"] = originalMetricType
		parse.SetPluginDimension(measurement, metricDims)
		parse.RemoveSFXDimensions(metricDims)

		// Get the metric value as a datapoint value
		if metricValue, err := datapoint.CastMetricValue(val); err == nil {
			var dp = datapoint.New(
				metricName,
				metricDims,
				metricValue.(datapoint.Value),
				metricType,
				GetTime(t...),
			)
			B.Output.SendDatapoint(dp)
		} else {
			// Skip if it's not an sfx event and it's not included
			if !isSFX && !B.Included(metricName) {
				continue
			}
			// We've already type checked field, so set property with value
			metricProps["message"] = val
			var ev = event.NewWithProperties(
				metricName,
				event.AGENT,
				metricDims,
				metricProps,
				GetTime(t...),
			)
			B.Output.SendEvent(ev)
		}
	}
}

// AddError handles errors reported to a telegraf accumulator
func (B *BaseEmitter) AddError(err error) {
	B.Logger.Error(err)
}

// NewEmitter returns a new BaseEmitter
func NewEmitter() (b *BaseEmitter) {
	b = &BaseEmitter{
		lock:      &sync.Mutex{},
		tagFilter: map[string]bool{},
		included:  map[string]bool{},
		excluded:  map[string]bool{},
	}
	return
}
