package monitors

// Filter of datapoints based on included status and user configuration of
// additionalMetrics.

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/utils/filter"
	"strings"
)

type metricsFilter struct {
	metadata          *Metadata
	additionalMetrics map[string]bool
	stringFilter      *filter.BasicStringFilter
}

func validateMetricName(metadata *Metadata, metricName string) error {
	if strings.TrimSpace(metricName) == "" {
		return errors.New("metricName cannot be empty")
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

		return fmt.Errorf("metric pattern '%s' did not match any available metrics", metricName)
	}

	if !metadata.HasMetric(metricName) {
		return errors.Errorf("metric '%s' does not exist", metricName)
	}

	return nil
}

func validateGroup(metadata *Metadata, group string) ([]string, error) {
	if strings.TrimSpace(group) == "" {
		return nil, errors.New("group cannot be empty")
	}

	metrics, ok := metadata.GroupMetricsMap[group]
	if !ok {
		return nil, errors.Errorf("group %s does not exist", group)
	}
	return metrics, nil
}

func newMetricsFilter(
	metadata *Metadata,
	additionalMetrics []config.AdditionalMetric) (*metricsFilter, error) {
	var filterItems []string

	for _, add := range additionalMetrics {
		switch {
		case add.MetricName == "" && add.Group == "":
			return nil, errors.New("either metricName or group must be set")
		case add.MetricName != "" && add.Group != "":
			return nil, errors.New("both metricName and group cannot be set")
		case add.Group != "":
			metrics, err := validateGroup(metadata, add.Group)
			if err != nil {
				return nil, err
			}
			filterItems = append(filterItems, metrics...)
		case add.MetricName != "":
			if err := validateMetricName(metadata, add.MetricName); err != nil {
				return nil, err
			}

			// If the user specified a metric that's already included no need to add it.
			if !metadata.IncludedMetrics[add.MetricName] {
				filterItems = append(filterItems, add.MetricName)
			}
		default:
			// Shouldn't be reached.
			return nil, fmt.Errorf("additionalMetric %v is invalid", add)
		}
	}

	basicFilter, err := filter.NewBasicStringFilter(filterItems)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to construct filter with items %s", filterItems)
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

	return &metricsFilter{metadata, effectiveMetrics, basicFilter}, nil
}

// enabledMetrics returns list of metrics that are enabled via user-configuration or by
// being included metrics.
func (add *metricsFilter) enabledMetrics() []string {
	metrics := make([]string, 0, len(add.additionalMetrics)+len(add.metadata.IncludedMetrics))

	for metric := range add.additionalMetrics {
		metrics = append(metrics, metric)
	}

	for metric := range add.metadata.IncludedMetrics {
		metrics = append(metrics, metric)
	}

	return metrics
}

func (add *metricsFilter) shouldSend(dp *datapoint.Datapoint) bool {
	if add.metadata.SendAll {
		return true
	}

	if add.metadata.HasIncludedMetric(dp.Metric) {
		// It's an included metric so send by default.
		return true
	}

	if add.additionalMetrics[dp.Metric] {
		// User has explicitly chosen to send this metric (or a group that this metric belongs to).
		return true
	}

	if add.metadata.MetricsExhaustive && !add.metadata.HasMetric(dp.Metric) {
		// Metrics list should be exhaustive but we don't know what this metric is.
		// so we drop it.
		return false
	}

	if !add.metadata.MetricsExhaustive && add.stringFilter.Matches(dp.Metric) {
		// If we reach here we don't know about the metric from the metadata
		// but it might match the filter. We have to check matches against
		// the filter because for unknown metrics it won't have an entry
		// in additionalMetrics.
		return true
	}

	// If we reach here the user hasn't enabled it in additionalMetrics.
	return false
}
