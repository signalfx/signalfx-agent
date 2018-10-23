// Package propfilters has logic describing the filtering of unwanted properties.  Filters
// are configured from the agent configuration file and is intended to be passed
// into each monitor for use if it sends propeties on its own.
package propfilters

import (
	"github.com/signalfx/signalfx-agent/internal/utils/filter"
)

// PropertyFilter can be used to filter out properties
type PropertyFilter interface {
	// Matches takes a property and returns whether it is matched by the
	// filter
	Matches(*datapoint.Datapoint) bool
}

// BasicPropertyFilter is designed to filter SignalFx property objects.  It
// can filter based on the monitor type, dimensions, or the metric name.  It
// supports both static, globbed, and regex patterns for filter values. If
// dimensions are specifed, they must all match for the properties to match. If
// multiple property names are given, only one must match for the property to
// match the filter since properties can only have one name.
type basicPropertyFilter struct {
	monitorType  string
	dimFilter    filter.StringMapFilter
	metricFilter filter.StringFilter
	negated      bool
}

// New returns a new filter with the given configuration
func New(monitorType string, metricNames []string, dimensions map[string]string, negated bool) (DatapointFilter, error) {
	var dimFilter filter.StringMapFilter
	if len(dimensions) > 0 {
		var err error
		dimFilter, err = filter.NewStringMapFilter(dimensions)
		if err != nil {
			return nil, err
		}
	}

	var metricFilter filter.StringFilter
	if len(metricNames) > 0 {
		var err error
		metricFilter, err = filter.NewStringFilter(metricNames)
		if err != nil {
			return nil, err
		}
	}

	return &basicPropertyFilter{
		monitorType:  monitorType,
		metricFilter: metricFilter,
		dimFilter:    dimFilter,
		negated:      negated,
	}, nil
}

// Matches tests a datapoint to see whether it is excluded by this filter.  In
// order to match on monitor type, the datapoint should have the "monitorType"
// key set in it's Meta field.
func (f *basicPropertyFilter) Matches(dp *datapoint.Datapoint) bool {
	if dpMonitorType, ok := dp.Meta[dpmeta.MonitorTypeMeta].(string); ok {
		if f.monitorType != "" && dpMonitorType != f.monitorType {
			return false
		}
	}

	matched := (f.metricFilter == nil || f.metricFilter.Matches(dp.Metric)) &&
		(f.dimFilter == nil || f.dimFilter.Matches(dp.Dimensions))

	if f.negated {
		return !matched
	}
	return matched
}
