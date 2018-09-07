package config

import "github.com/signalfx/signalfx-agent/internal/core/dpfilters"

// MetricFilter describes a set of subtractive filters applied to datapoints
// right before they are sent.
type MetricFilter struct {
	// A map of dimension key/values to match against.  All key/values must
	// match a datapoint for it to be matched.
	Dimensions map[string]string `yaml:"dimensions" default:"{}"`
	// A list of metric names to match against, OR'd together
	MetricNames []string `yaml:"metricNames"`
	// A single metric name to match against
	MetricName string `yaml:"metricName"`
	// (**Only applicable for the top level filters**) Limits this scope of the
	// filter to datapoints from a specific monitor. If specified, any
	// datapoints not from this monitor type will never match against this
	// filter.
	MonitorType string `yaml:"monitorType"`
	// Negates the result of the match so that it matches all datapoints that
	// do NOT match the metric name and dimension values given. This does not
	// negate monitorType, if given.
	Negated bool `yaml:"negated"`
}

// MakeFilter returns an actual filter instance from the config
func (mf *MetricFilter) MakeFilter() (dpfilters.DatapointFilter, error) {
	if mf.MetricName != "" {
		mf.MetricNames = append(mf.MetricNames, mf.MetricName)
	}
	return dpfilters.New(mf.MonitorType, mf.MetricNames, mf.Dimensions, mf.Negated)
}

func makeFilterSet(conf []MetricFilter) (*dpfilters.FilterSet, error) {
	fs := make([]dpfilters.DatapointFilter, 0)
	mtes := make([]MetricFilter, 0, len(conf))

	for _, mte := range conf {
		mtes = AddOrMerge(mtes, mte)
	}

	for _, mte := range mtes {
		f, err := mte.MakeFilter()
		if err != nil {
			return nil, err
		}
		fs = append(fs, f)
	}

	return &dpfilters.FilterSet{
		Filters: fs,
	}, nil
}

// AddOrMerge MetricFilter to list or merge with existing MetricFilter
func AddOrMerge(mtes []MetricFilter, mf2 MetricFilter) []MetricFilter {
	for i, mf1 := range mtes {
		if mf1.ShouldMerge(mf2) {
			mtes[i] = mf1.MergeWith(mf2)
			return mtes
		}
	}
	return append(mtes, mf2)
}

// MergeWith merges mf2's MetricFilter.MetricNames into receiver mf MetricFilter.MetricNames
func (mf *MetricFilter) MergeWith(mf2 MetricFilter) MetricFilter {
	if mf2.MetricName != "" {
		mf2.MetricNames = append(mf2.MetricNames, mf2.MetricName)
	}
	for _, metricName := range mf2.MetricNames {
		mf.MetricNames = append(mf.MetricNames, metricName)
	}
	return *mf
}

// ShouldMerge checks if mf2 MetricFilter should be merged into receiver mf MetricFilter
// Filters with same monitorType, negation, and dimensions should be merged
func (mf *MetricFilter) ShouldMerge(mf2 MetricFilter) bool {
	if mf.MonitorType != mf2.MonitorType {
		return false
	}
	if mf.Negated != mf2.Negated {
		return false
	}
	if len(mf.Dimensions) != len(mf2.Dimensions) {
		return false
	}
	// Ensure no differing dimension values
	for k, v := range mf.Dimensions {
		if mf2.Dimensions[k] != v {
			return false
		}
	}
	return true
}
