package monitors

import (
	"fmt"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/core/dpfilters"
)

type monitorFiltering struct {
	filterSet *dpfilters.FilterSet
}

// AddDatapointExclusionFilter to the monitor's filter set.  Make sure you do this
// before any datapoints are sent as it is not thread-safe with SendDatapoint.
func (mf *monitorFiltering) AddDatapointExclusionFilter(filter dpfilters.DatapointFilter) {
	mf.filterSet.ExcludeFilters = append(mf.filterSet.ExcludeFilters, filter)
}

func (mf *monitorFiltering) EnabledMetrics() []string {
	dp := &datapoint.Datapoint{}
	var enabledMetrics []string

	for metric := range mf.metadata.Metrics {
		dp.Metric = metric
		if !filterSet.Matches(dp) {
			enabledMetrics = append(enabledMetrics, metric)
		}
	}
}

func buildFilterSet(metadata *Metadata, conf config.MonitorCustomConfig) (*dpfilters.FilterSet, []string, error) {
	coreConfig := conf.MonitorConfigCore()

	oldFilter, err := coreConfig.OldFilterSet()
	if err != nil {
		return nil, nil, err
	}

	newFilter, err := coreConfig.NewFilterSet()
	if err != nil {
		return nil, nil, err
	}

	excludeFilters := []dpfilters.DatapointFilter{oldFilter, newFilter}

	if !metadata.SendAll {
		// Make a copy of extra metrics from config so we don't alter what the user configured.
		extraMetrics := append([]string{}, coreConfig.ExtraMetrics...)

		// Monitors can add additional extra metrics to allow through such as based on config flags.
		if monitorExtra, ok := conf.(config.ExtraMetrics); ok {
			extraMetrics = append(extraMetrics, monitorExtra.GetExtraMetrics()...)
		}

		includedMetricsFilter, err := newMetricsFilter(metadata, extraMetrics, coreConfig.ExtraGroups)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to construct extraMetrics filter: %s", err)
		}

		// Prepend the included metrics filter.
		excludeFilters = append([]dpfilters.DatapointFilter{dpfilters.Negate(includedMetricsFilter)}, excludeFilters...)
	}

	filterSet := &dpfilters.FilterSet{
		ExcludeFilters: excludeFilters,
	}

	return filterSet, enabledMetrics, nil
}
