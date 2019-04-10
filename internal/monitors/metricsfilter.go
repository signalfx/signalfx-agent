package monitors

// Filter of datapoints based on included status and user configuration of
// extraMetrics and extraGroups.

import (
	"errors"
	"fmt"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/dpfilters"
	"github.com/signalfx/signalfx-agent/internal/utils/filter"
	"github.com/sirupsen/logrus"
	"strings"
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

		logrus.Warnf("extraMetrics: metric pattern '%s' did not match any available metrics", metricName)
	}

	if !metadata.HasMetric(metricName) {
		logrus.Warnf("extraMetrics: metric '%s' does not exist", metricName)
	}

	return nil
}

func validateGroup(metadata *Metadata, group string) ([]string, error) {
	if strings.TrimSpace(group) == "" {
		return nil, errors.New("group cannot be empty")
	}

	metrics, ok := metadata.GroupMetricsMap[group]
	if !ok {
		return nil, fmt.Errorf("group %s does not exist", group)
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

// enabledMetrics returns list of metrics that are enabled via user-configuration or by
// being included metrics.
func (mf *extraMetricsFilter) enabledMetrics() []string {
	metrics := make([]string, 0, len(mf.extraMetrics)+len(mf.metadata.IncludedMetrics))

	for metric := range mf.extraMetrics {
		metrics = append(metrics, metric)
	}

	for metric := range mf.metadata.IncludedMetrics {
		metrics = append(metrics, metric)
	}

	return metrics
}

func (mf *extraMetricsFilter) Matches(dp *datapoint.Datapoint) bool {
	if mf.metadata.SendAll {
		return true
	}

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
