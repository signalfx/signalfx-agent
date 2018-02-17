package config

import "github.com/signalfx/signalfx-agent/internal/core/filters"

// MetricFilter describes a set of subtractive filters applied to datapoints
// right before they are sent.
type MetricFilter struct {
	Dimensions  map[string]string `yaml:"dimensions,omitempty" default:"{}"`
	MetricNames []string          `yaml:"metricNames,omitempty"`
	MetricName  string            `yaml:"metricName,omitempty"`
	MonitorType string            `yaml:"monitorType,omitempty"`
	Negated     bool              `yaml:"negated,omitempty"`
}

// Help provides documentation for this config struct's fields
func (mf *MetricFilter) Help() map[string]string {
	return map[string]string{
		"Dimensions":  "A map of dimension key/values to match against.  All key/values must match a datapoint for it to be matched.",
		"MetricNames": "A list of metric names to match against, OR'd together",
		"MetricName":  "A single metric name to match against",
		"MonitorType": "Limits this filter to datapoints from a specific monitor",
		"Negated":     "Negates the result of the match so that it matches all datapoints that do NOT match the non-negated filter.",
	}
}

// MakeFilter returns an actual filter instance from the config
func (mf *MetricFilter) MakeFilter() filters.Filter {
	if mf.MetricName != "" {
		mf.MetricNames = append(mf.MetricNames, mf.MetricName)
	}
	return filters.New(mf.MonitorType, mf.MetricNames, mf.Dimensions, mf.Negated)
}
