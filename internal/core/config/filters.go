package config

import "github.com/signalfx/signalfx-agent/internal/core/filters"

// MetricFilter describes a set of subtractive filters applied to datapoints
// right before they are sent.
type MetricFilter struct {
	// A map of dimension key/values to match against.  All key/values must
	// match a datapoint for it to be matched.
	Dimensions map[string]string `yaml:"dimensions,omitempty" default:"{}"`
	// A list of metric names to match against, OR'd together
	MetricNames []string `yaml:"metricNames,omitempty"`
	// A single metric name to match against
	MetricName string `yaml:"metricName,omitempty"`
	// Limits this scope of the filter to datapoints from a specific monitor.
	// If specified, any datapoints not from this monitor type will never match
	// against this filter.
	MonitorType string `yaml:"monitorType,omitempty"`
	// Negates the result of the match so that it matches all datapoints that
	// do NOT match the metric name and dimension values given. This does not
	// negate monitorType, if given.
	Negated bool `yaml:"negated,omitempty"`
}

// MakeFilter returns an actual filter instance from the config
func (mf *MetricFilter) MakeFilter() filters.Filter {
	if mf.MetricName != "" {
		mf.MetricNames = append(mf.MetricNames, mf.MetricName)
	}
	return filters.New(mf.MonitorType, mf.MetricNames, mf.Dimensions, mf.Negated)
}
