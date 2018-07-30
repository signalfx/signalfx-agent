// +build !windows

package kafkaproducer

import (
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/genericjmx"
	yaml "gopkg.in/yaml.v2"
)

const monitorType = "collectd/kafka_producer"

// MONITOR(collectd/kafka_producer): Monitors a Java based Kafka producer using GenericJMX.
//
// See the [integration documentation](https://github.com/signalfx/integrations/tree/master/collectd-kafka)
// for more information.
//
// This monitor has a set of [built in MBeans
// configured](https://github.com/signalfx/signalfx-agent/tree/master/internal/monitors/collectd/kafkaproducer/mbeans.go)
// for which it pulls metrics from the Kafka producer's JMX endpoint.
//
// Sample YAML configuration:
//```yaml
// monitors:
//   - type: collectd/kafka_producer
//     host: localhost
//     port: 8099
// ```
//
// Note that this monitor requires Kafka v0.9.0.0 or above and collects metrics from the new producer API.

// GAUGE(kafka.producer.response-rate): Average number of responses received per second.

// GAUGE(kafka.producer.request-rate): Average number of requests sent per second.

// GAUGE(kafka.producer.request-latency-avg): Average request latency in ms. Time it takes on average for the producer to get
// responses from the broker

// GAUGE(kafka.producer.outgoing-byte-rate): Average number of outgoing bytes sent per second to all servers.

// GAUAGE(kafka.producer.io-wait-time-ns-avg): Average length of time the I/O thread spent waiting for a socket ready for
// reads or writes in nanoseconds

// GAUAGE(kafka.producer.byte-rate): Average number of bytes sent per second for a topic.

// GAUAGE(kafka.producer.compression-rate): Average compression rate of record batches for a topic.

// GAUAGE(kafka.producer.record-error-rate): Average per-second number of record sends that resulted in errors for a topic.

// GAUAGE(kafka.producer.record-retry-rate): Average per-second number of retried record sends for a topic.

// GAUAGE(kafka.producer.record-send-rate): Average number of records sent per second for a topic.


var serviceName = "kafka_producer"

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
