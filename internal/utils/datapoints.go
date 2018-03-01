package utils

import (
	"fmt"
	"sort"
	"strings"

	"github.com/signalfx/golib/datapoint"
)

func sortedDimensionString(dims map[string]string) string {
	var keys []string
	for k := range dims {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var dimStrs []string
	for _, k := range keys {
		dimStrs = append(dimStrs, fmt.Sprintf("%s=%s", k, dims[k]))
	}

	return strings.Join(dimStrs, "; ")
}

func dpTypeToString(t datapoint.MetricType) string {
	switch t {
	case datapoint.Gauge:
		return "gauge"
	case datapoint.Count:
		return "counter"
	case datapoint.Counter:
		return "cumulative counter"
	default:
		return fmt.Sprintf("unsupported type %d", t)
	}
}

// DatapointToString pretty prints a datapoint in a consistent manner for
// logging purposes.  The most important thing here is to sort the dimension
// dict so it is consistent so that it is easier to visually scan a large list
// of datapoints.
func DatapointToString(dp *datapoint.Datapoint) string {
	return fmt.Sprintf("%s: %s (%s) @ %s\n[%s]", dp.Metric, dp.Value, dpTypeToString(dp.MetricType), dp.Timestamp, sortedDimensionString(dp.Dimensions))
}

// BoolToInt returns 1 if b is true and 0 otherwise.  It is useful for
// datapoints which track a binary value since we don't support boolean
// datapoint directly.
func BoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
