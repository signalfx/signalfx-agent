package config

import "github.com/signalfx/signalfx-agent/internal/core/filters"

// MetricFilter describes a set of subtractive filters applied to datapoints
// right before they are sent.
type MetricFilter struct {
	Dimensions  map[string]string `yaml:"dimensions,omitempty" default:"{}"`
	MetricNames []string          `yaml:"metricNames,omitempty"`
	MetricName  string            `yaml:"metricName,omitempty"`
	MonitorType string            `yaml:"monitorType,omitempty"`
}

// MakeFilter returns an actual filter instance from the config
func (mf *MetricFilter) MakeFilter() filters.Filter {
	if mf.MetricName != "" {
		mf.MetricNames = append(mf.MetricNames, mf.MetricName)
	}
	return filters.New(mf.MonitorType, mf.MetricNames, mf.Dimensions)
}
