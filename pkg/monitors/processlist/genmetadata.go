// Code generated by monitor-code-gen. DO NOT EDIT.

package processlist

import (
	"github.com/signalfx/signalfx-agent/pkg/monitors"
)

const monitorType = "processlist"

var groupSet = map[string]bool{}

var metricSet = map[string]monitors.MetricInfo{}

var defaultMetrics = map[string]bool{}

var groupMetricsMap = map[string][]string{}

var monitorMetadata = monitors.Metadata{
	MonitorType:       "processlist",
	DefaultMetrics:    defaultMetrics,
	Metrics:           metricSet,
	MetricsExhaustive: false,
	Groups:            groupSet,
	GroupMetricsMap:   groupMetricsMap,
	SendAll:           false,
}
