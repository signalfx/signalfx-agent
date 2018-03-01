package cassandra

import (
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/genericjmx"
	yaml "gopkg.in/yaml.v2"
)

const monitorType = "collectd/cassandra"

// MONITOR(collectd/cassandra): Monitors Cassandra using the GenericJMX collectd
// plugin.

var serviceName = "cassandra"

// Monitor is the main type that represents the monitor
type Monitor struct {
	*genericjmx.JMXMonitorCore
}

func init() {
	var defaultMBeans genericjmx.MBeanMap
	err := yaml.Unmarshal([]byte(defaultMBeanYAML), &defaultMBeans)
	if err != nil {
		panic("YAML for GenericJMX MBeans is invalid: " + err.Error())
	}
	defaultMBeans = defaultMBeans.MergeWith(genericjmx.DefaultMBeans)

	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			genericjmx.NewJMXMonitorCore(defaultMBeans, serviceName),
		}
	}, &genericjmx.Config{})
}

// CUMULATIVE(counter.cassandra.ClientRequest.RangeSlice.Latency.Count): Count
// of range slice operations since server start

// CUMULATIVE(counter.cassandra.ClientRequest.RangeSlice.Timeouts.Count): Count
// of range slice timeouts since server start

// CUMULATIVE(counter.cassandra.ClientRequest.RangeSlice.Unavailables.Count):
// Count of range slice unavailables since server start

// CUMULATIVE(counter.cassandra.ClientRequest.Read.Latency.Count): Count of
// read operations since server start

// CUMULATIVE(counter.cassandra.ClientRequest.Read.Timeouts.Count): Count of
// read timeouts since server start

// CUMULATIVE(counter.cassandra.ClientRequest.Read.Unavailables.Count): Count
// of read unavailables since server start

// CUMULATIVE(counter.cassandra.ClientRequest.Write.Latency.Count): Count of
// write operations since server start

// CUMULATIVE(counter.cassandra.ClientRequest.Write.Timeouts.Count): Count of
// write timeouts since server start

// CUMULATIVE(counter.cassandra.ClientRequest.Write.Unavailables.Count): Count
// of write unavailables since server start

// CUMULATIVE(counter.cassandra.Compaction.TotalCompactionsCompleted.Count):
// Number of compaction operations since node start

// GAUGE(gauge.cassandra.ClientRequest.RangeSlice.Latency.50thPercentile): 50th
// percentile (median) of Cassandra range slice latency

// GAUGE(gauge.cassandra.ClientRequest.RangeSlice.Latency.99thPercentile): 99th
// percentile of Cassandra range slice latency

// GAUGE(gauge.cassandra.ClientRequest.RangeSlice.Latency.Max): Maximum
// Cassandra range slice latency

// GAUGE(gauge.cassandra.ClientRequest.Read.Latency.50thPercentile): 50th
// percentile (median) of Cassandra read latency

// GAUGE(gauge.cassandra.ClientRequest.Read.Latency.99thPercentile): 99th
// percentile of Cassandra read latency

// GAUGE(gauge.cassandra.ClientRequest.Read.Latency.Max): Maximum Cassandra
// read latency

// GAUGE(gauge.cassandra.ClientRequest.Write.Latency.50thPercentile): 50th
// percentile (median) of Cassandra write latency

// GAUGE(gauge.cassandra.ClientRequest.Write.Latency.99thPercentile): 99th
// percentile of Cassandra write latency

// GAUGE(gauge.cassandra.ClientRequest.Write.Latency.Max): Maximum Cassandra
// write latency

// GAUGE(gauge.cassandra.Compaction.PendingTasks.Value): Number of compaction
// operations waiting to run

// GAUGE(gauge.cassandra.Storage.Load.Count): Storage used for Cassandra data
// in bytes

// GAUGE(gauge.cassandra.Storage.TotalHints.Count): Total hints since node
// start

// GAUGE(gauge.cassandra.Storage.TotalHintsInProgress.Count): Total pending
// hints
