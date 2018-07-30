// +build !windows

package kafka

import (
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/genericjmx"
	yaml "gopkg.in/yaml.v2"
)

const monitorType = "collectd/kafka"

var serviceName = "kafka"

// MONITOR(collectd/kafka): Monitors a Kafka instance using collectd's
// GenericJMX plugin.
//
// This monitor has a set of [built in MBeans
// configured](https://github.com/signalfx/signalfx-agent/tree/master/internal/monitors/collectd/kafka/mbeans.go)
// for which it pulls metrics from Kafka's JMX endpoint.
//
// Note that this monitor supports Kafka v0.8.2.x and above. For Kafka v1.x.x and above,
// apart from the list of default metrics, kafka.server:type=ZooKeeperClientMetrics,name=ZooKeeperRequestLatencyMs
// is a good metric to monitor since it gives an understanding of how long brokers wait for
// requests to Zookeeper to be completed. Since Zookeeper is an integral part of a Kafka cluster,
// monitoring it using the [Zookeeper
// monitor] (https://docs.signalfx.com/en/latest/integrations/agent/monitors/collectd-zookeeper.html)
// is recommended. It is also a good idea to monitor disk utilization and network metrics of
// the underlying host.
//
// See https://github.com/signalfx/integrations/tree/master/collectd-kafka.

// CUMULATIVE(counter.kafka-all-bytes-in): Number of bytes received per second
// across all topics

// CUMULATIVE(counter.kafka-all-bytes-out): Number of bytes transmitted per
// second across all topics

// CUMULATIVE(counter.kafka-log-flushes): Number of log flushes per second

// CUMULATIVE(counter.kafka-messages-in): Number of messages received per
// second across all topics

// CUMULATIVE(counter.kafka.fetch-consumer.total-time.count): Number of fetch
// requests from consumers per second across all partitions

// CUMULATIVE(counter.kafka.fetch-follower.total-time.count): Number of fetch
// requests from followers per second across all partitions

// GAUGE(counter.kafka.produce.total-time.99th): 99th percentile of time in
// milliseconds to process produce requests

// CUMULATIVE(counter.kafka.produce.total-time.count): Number of producer
// requests

// GAUGE(counter.kafka.produce.total-time.median): Median time it takes to
// process a produce request

// GAUGE(gauge.kafka-active-controllers): Specifies if the broker an active
// controller

// GAUGE(gauge.kafka-log-flush-time-ms-p95): 95th percentile of log flush time
// in milliseconds

// GAUGE(gauge.kafka-log-flush-time-ms): Average number of milliseconds to
// flush a log

// GAUGE(gauge.kafka-request-queue): Number of requests in the request queue
// across all partitions on the broker

// GAUGE(gauge.kafka-underreplicated-partitions): Number of underreplicated
// partitions across all topics on the broker

// GAUGE(gauge.kafka.fetch-consumer.total-time.99th): 99th percentile of time
// in milliseconds to process fetch requests from consumers

// GAUGE(gauge.kafka.fetch-consumer.total-time.median): Median time it takes to
// process a fetch request from consumers

// GAUGE(gauge.kafka.fetch-follower.total-time.99th): 99th percentile of time
// in milliseconds to process fetch requests from followers

// GAUGE(gauge.kafka.fetch-follower.total-time.median): Median time it takes to
// process a fetch request from follower

// GAUGE(kafka-offline-partitions-count): Number of partitions that don’t have an
// active leader and are hence not writable or readable

// CUMULATIVE(kafka-isr-shrinks): When a broker goes down, ISR for some of partitions
// will shrink. When that broker is up again, ISR will be expanded once the replicas
// are fully caught up. Other than that, the expected value for both ISR shrink rate
// and expansion rate is 0.

// CUMULATIVE(kafka-isr-expands): When a broker is brought up after a failure, it starts
// catching up by reading from the leader. Once it is caught up, it gets added back
//to the ISR.

// GAUGE(kafka-max-lag): Maximum lag in messages between the follower and leader replicas

// CUMULATIVE(kafka-leader-election-rate): Number of leader elections

// CUMULATIVE(kafka-unclean-elections): Number of unclean leader elections. This happens
// when a leader goes down and an out-of-sync replica is chosen to be the leader

// Monitor is the main type that represents the monitor
type Monitor struct {
	*genericjmx.JMXMonitorCore
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	genericjmx.Config `yaml:",inline"`
	// Cluster name to which the broker belongs
	ClusterName string `yaml:"clusterName" validate:"required"`
}

// Configure configures the kafka monitor and instantiates the generic jmx
// monitor
func (m *Monitor) Configure(conf *Config) error {
	m.Output.AddExtraDimension("cluster", conf.ClusterName)
	return m.JMXMonitorCore.Configure(&conf.Config)
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
	}, &Config{})
}
