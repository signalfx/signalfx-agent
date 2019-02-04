package baseemitter

import (
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	measure "github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/measurement"
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
	Output types.Output
	Logger log.FieldLogger
	// omittedTags are tags that should be removed from measurements before
	// being processed
	omittedTags map[string]bool
	// addTags are tags that should be added to all measurements
	addTags map[string]string
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
	// name map is a map of metric names to their desired metricname
	// this is used for overriding metric names
	nameMap map[string]string
	// metricNameTransformations is an array of functions to apply to parsed metric name
	// from a telegraf metric.
	metricNameTransformations []func(metricName string) string
	// measurementTransformations is an array of functions to apply to an incoming measurement
	// before retrieving the metric name, checking for inclusion/exclusion, etc.
	// Use great discretion with this.
	measurementTransformations []func(*measure.Measurement) error
	// whether to omit the "telegraf_type"
	// dimension for documenting original metric type
	omitOriginalMetricType bool
}

// AddTag adds a key/value pair to all measurement tags.  If a key conflicts
// the key value pair in AddTag will override the original key on the
// measurement
func (b *BaseEmitter) AddTag(key string, val string) {
	b.addTags[key] = val
}

// AddTags adds a map of key value pairs to all measurement tags.  If a key
// conflicts the key value pair in AddTags will override the original key on
// the measurement.
func (b *BaseEmitter) AddTags(tags map[string]string) {
	for k, v := range tags {
		b.AddTag(k, v)
	}
}

// IncludeEvent registers an event name to include
// during emission. We disable all events by default because Telegraf has some
// junk events.
func (b *BaseEmitter) IncludeEvent(name string) {
	b.included[name] = true
}

// IncludeEvents registers a list of event names to
// include during emission. We disable all events by default because Telegraf
// has some junk events.
func (b *BaseEmitter) IncludeEvents(names []string) {
	for _, name := range names {
		b.IncludeEvent(name)
	}
}

// Included - checks if events should be included
// during emission.  We disable all events by default because Telegraf has some
// junk events.
func (b *BaseEmitter) Included(name string) bool {
	return b.included[name]
}

// ExcludeDatum adds a name to the list of metrics and events to
// exclude
func (b *BaseEmitter) ExcludeDatum(name string) {
	b.excluded[name] = true
}

// ExcludeData adds a list of names the list of metrics and events
// to exclude
func (b *BaseEmitter) ExcludeData(names []string) {
	for _, name := range names {
		b.ExcludeDatum(name)
	}
}

// IsExcluded - checks if events or metrics should be
// excluded from emission
func (b *BaseEmitter) IsExcluded(name string) bool {
	return b.excluded[name]
}

// OmitTag adds a tag to the list of tags to remove from measurements
func (b *BaseEmitter) OmitTag(tag string) {
	b.omittedTags[tag] = true
}

// OmitTags adds a list of tags the list of tags to remove from measurements
func (b *BaseEmitter) OmitTags(tags []string) {
	for _, tag := range tags {
		b.OmitTag(tag)
	}
}

// FilterTags - filter function for util.CloneAndFilterStringMapWithFunc()
// it returns true if the supplied key is not in the omittedTags map
func (b *BaseEmitter) FilterTags(key string, value string) bool {
	return !b.omittedTags[key]
}

// RenameMetric adds a mapping to rename a metric by it's name
func (b *BaseEmitter) RenameMetric(original string, override string) {
	b.nameMap[original] = override
}

// RenameMetrics takes a map of metric name overrides map[original]override
func (b *BaseEmitter) RenameMetrics(mappings map[string]string) {
	for original, override := range mappings {
		b.RenameMetric(original, override)
	}
}

// GetMetricName parses the metric name and takes name overrides into account
// if a name is overridden it will not have transformations applied to it
func (b *BaseEmitter) GetMetricName(measurement string, field string, metricDims map[string]string) (string, bool) {
	var name, isSFX = parse.GetMetricName(measurement, field, metricDims)

	if altName := b.nameMap[name]; altName != "" {
		return altName, isSFX
	}

	// apply metricname transformations
	for _, f := range b.metricNameTransformations {
		name = f(name)
	}

	return name, isSFX
}

// AddMetricNameTransformation adds a function for mutating metric names.  GetMetricNames()
// will invoke each of the transformation functions after the metric name is parsed
// from the incoming measurement.
func (b *BaseEmitter) AddMetricNameTransformation(f func(string) string) {
	b.metricNameTransformations = append(b.metricNameTransformations, f)
}

// AddMetricNameTransformations adds a list of functions for mutating metric names.  GetMetricNames()
// will invoke each of the transformation functions after the metric name is parsed
// from the incoming measurement.
func (b *BaseEmitter) AddMetricNameTransformations(fns []func(string) string) {
	for _, f := range fns {
		b.AddMetricNameTransformation(f)
	}
}

// AddMeasurementTransformation adds a function to the list of functions the emitter
// will pass an incoming measurement through.  This is useful for manipulating tags
// and fields before the measurement is converted to a SignalFx datapoint.
func (b *BaseEmitter) AddMeasurementTransformation(f func(*measure.Measurement) error) {
	b.measurementTransformations = append(b.measurementTransformations, f)
}

// AddMeasurementTransformations a list of functions to the list of functions the emitter
// will pass an incoming measurement through.  This is useful for manipulating tags
// and fields before the measurement is converted to a SignalFx datapoint.
func (b *BaseEmitter) AddMeasurementTransformations(fns []func(*measure.Measurement) error) {
	for _, f := range fns {
		b.AddMeasurementTransformation(f)
	}
}

// TransformMeasurement applies all measurementTransformations to the supplied measurement
func (b *BaseEmitter) TransformMeasurement(m *measure.Measurement) {
	// apply transformation functions to incoming measurement
	for _, tf := range b.measurementTransformations {
		if err := tf(m); err != nil {
			b.Logger.WithError(err).Errorf("an error occurred applying a transformation to the measurement %v", m)
		}
	}
}

// Add parses measurements from telegraf and emits them through Output
func (b *BaseEmitter) Add(measurement string, fields map[string]interface{},
	tags map[string]string, metricType datapoint.MetricType,
	originalMetricType string, t ...time.Time) {

	// create a measurement
	// telegraf doc says that tags are owned by the calling plugin and they
	// shouldn't be mutated.  So we copy the tags map
	ms := measure.New(measurement, fields, utils.CloneStringMap(tags), metricType, originalMetricType, t...)

	// apply transformation functions to the measurement
	b.TransformMeasurement(ms)

	for field, val := range ms.Fields {
		metricDims := utils.CloneAndFilterStringMapWithFunc(ms.Tags, b.FilterTags)

		// add additional tags to the metricDims
		if len(b.addTags) > 0 {
			metricDims = utils.MergeStringMaps(metricDims, b.addTags)
		}

		// Generate the metric name
		metricName, isSFX := b.GetMetricName(ms.Measurement, field, metricDims)

		// Check if the metric is explicitly excluded
		if b.IsExcluded(metricName) {
			b.Logger.Debugf("excluding the following metric: %s", metricName)
			continue
		}

		// If eligible, move the dimension "property" to properties
		metricProps, propErr := parse.ExtractProperty(metricName, metricDims)
		if propErr != nil {
			b.Logger.Error(propErr)
			continue
		}

		// Add common dimensions
		if originalMetricType != "" && !b.omitOriginalMetricType {
			// only add telegraf_type if we override the original type
			metricDims["telegraf_type"] = originalMetricType
		}
		parse.SetPluginDimension(ms.Measurement, metricDims)
		parse.RemoveSFXDimensions(metricDims)

		// Get the metric value as a datapoint value
		if metricValue, err := datapoint.CastMetricValue(val); err == nil {
			var dp = datapoint.New(
				metricName,
				metricDims,
				metricValue,
				metricType,
				GetTime(ms.Timestamps...),
			)
			b.Output.SendDatapoint(dp)
		} else {
			// Skip if it's not an sfx event and it's not included
			if !isSFX && !b.Included(metricName) {
				continue
			}
			// We've already type checked field, so set property with value
			metricProps["message"] = val
			var ev = event.NewWithProperties(
				metricName,
				event.AGENT,
				metricDims,
				metricProps,
				GetTime(ms.Timestamps...),
			)
			b.Output.SendEvent(ev)
		}
	}
}

// AddError handles errors reported to a telegraf accumulator
func (b *BaseEmitter) AddError(err error) {
	// some telegraf plugins will invoke AddError with nil i.e. sqlserver
	if err != nil {
		b.Logger.WithError(err).Errorf("an error was emitted from the plugin")
	}
}

// SetOmitOrignalMetricType accepts a boolean to indicate whether the emitter should
// add the original metric type or not to each metric
func (b *BaseEmitter) SetOmitOrignalMetricType(in bool) {
	b.omitOriginalMetricType = in
}

// NewEmitter returns a new BaseEmitter
func NewEmitter(Output types.Output, Logger log.FieldLogger) *BaseEmitter {
	return &BaseEmitter{
		Output:                     Output,
		Logger:                     Logger,
		omittedTags:                map[string]bool{},
		included:                   map[string]bool{},
		excluded:                   map[string]bool{},
		addTags:                    map[string]string{},
		nameMap:                    map[string]string{},
		metricNameTransformations:  []func(string) string{},
		measurementTransformations: []func(*measure.Measurement) error{},
	}
}
