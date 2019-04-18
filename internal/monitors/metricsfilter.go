package monitors

// Filter of datapoints based on included status and user configuration of
// extraMetrics and extraGroups.

import (
	"errors"

	"fmt"
	"strings"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/dpfilters"
	"github.com/signalfx/signalfx-agent/internal/utils/filter"
	"github.com/sirupsen/logrus"
)

var _ dpfilters.DatapointFilter = &extraMetricsFilter{}

type extraMetricsFilter struct {
	metadata     *Metadata
	extraMetrics map[string]bool
	stringFilter *filter.BasicStringFilter
}

func validateMetricName(metadata *Metadata, metricName string) error {
	if strings.TrimSpace(metricName) == "" {
		return errors.New("metric name cannot be empty")
	}

	if !metadata.MetricsExhaustive {
		// The metrics list isn't exhaustive so can't do extra validation.
		return nil
	}

	if strings.ContainsRune(metricName, '*') {
		// Make sure a metric pattern matches at least one metric.
		m, err := filter.NewBasicStringFilter([]string{metricName})
		if err != nil {
			return err
		}

		for metric := range metadata.Metrics {
			if m.Matches(metric) {
				return nil
			}
		}

		logrus.Warnf("extraMetrics: metric pattern '%s' did not match any available metrics for monitor %s",
			metricName, metadata.MonitorType)
	}

	if !metadata.HasMetric(metricName) {
		logrus.Warnf("extraMetrics: metric '%s' does not exist for monitor %s", metricName, metadata.MonitorType)
	}

	return nil
}

func validateGroup(metadata *Metadata, group string) ([]string, error) {
	if strings.TrimSpace(group) == "" {
		return nil, errors.New("group cannot be empty")
	}

	metrics, ok := metadata.GroupMetricsMap[group]
	if !ok {
		logrus.Warnf("extraMetrics: group %s does not exist for monitor %s", group, metadata.MonitorType)
	}
	return metrics, nil
}

func newMetricsFilter(metadata *Metadata, extraMetrics, extraGroups []string) (*extraMetricsFilter, error) {
	var filterItems []string

	for _, metric := range extraMetrics {
		if err := validateMetricName(metadata, metric); err != nil {
			return nil, err
		}

		// If the user specified a metric that's already included no need to add it.
		if !metadata.IncludedMetrics[metric] {
			filterItems = append(filterItems, metric)
		}
	}

	for _, group := range extraGroups {
		metrics, err := validateGroup(metadata, group)
		if err != nil {
			return nil, err
		}
		filterItems = append(filterItems, metrics...)
	}

	basicFilter, err := filter.NewBasicStringFilter(filterItems)
	if err != nil {
		return nil, fmt.Errorf("unable to construct filter with items %s: %s", filterItems, err)
	}

	effectiveMetrics := map[string]bool{}

	// Precompute set of metrics that matches the filter. This isn't a complete
	// set of metrics that are enabled in the case of metrics that aren't included
	// in metadata. But it provides a fast path for known metrics.
	for metric := range metadata.Metrics {
		if basicFilter.Matches(metric) {
			effectiveMetrics[metric] = true
		}
	}

	return &extraMetricsFilter{metadata, effectiveMetrics, basicFilter}, nil
}

func (mf *extraMetricsFilter) Matches(dp *datapoint.Datapoint) bool {
	if mf.metadata.HasIncludedMetric(dp.Metric) {
		// It's an included metric so send by default.
		return true
	}

	if mf.extraMetrics[dp.Metric] {
		// User has explicitly chosen to send this metric (or a group that this metric belongs to).
		return true
	}

	// Lastly check if it matches filter. If it's a known metric from metadata will get matched
	// above so this is a last check for unknown metrics.
	return mf.stringFilter.Matches(dp.Metric)
}
