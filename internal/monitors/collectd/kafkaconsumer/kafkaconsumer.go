// +build !windows

package kafkaconsumer

import (
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/genericjmx"
	yaml "gopkg.in/yaml.v2"
)

const monitorType = "collectd/kafka_consumer"

// MONITOR(collectd/kafka_consumer): Monitors a Java based Kafka consumer using GenericJMX.
//
// See the [integration documentation](https://github.com/signalfx/integrations/tree/master/collectd-kafka_consumer)
// for more information.
//
// This monitor has a set of [built in MBeans
// configured](https://github.com/signalfx/signalfx-agent/tree/master/internal/monitors/collectd/kafka_consumer/mbeans.go)
// for which it pulls metrics from the Kafka consumer's JMX endpoint.
//
// Sample YAML configuration:
//```yaml
// monitors:
//   - type: collectd/kafka_consumer
//     host: localhost
//     port: 9099
//     mBeansToOmit:
//       - fetch-size-avg-per-topic
// ```
//
// Note that this monitor requires Kafka v0.9.0.0 or above and collects metrics from the new consumer API.
// Also, per topic metrics that are collected by default are not available through the new consumer API in
// v0.9.0.0 which can cause the logs to flood with warnings related to MBean not being found.
// Use `mBeansToOmit` config option in such cases. The above example configuration will not attempt to
// collect the MBean referenced by `fetch-size-avg-per-topic`. Here is a
// [list] (https://github.com/signalfx/signalfx-agent/tree/master/internal/monitors/collectd/kafka/mbeans.go)
// of metrics collected by default

// GAUGE(kafka.consumer.records-lag-max): Maximum lag in of records for any partition in this window. An increasing
// value over time is your best indication that the consumer group is not keeping up with the producers.

// GAUGE(kafka.consumer.bytes-consumed-rate): Average number of bytes consumed per second. This metric has either
// client-id dimension or, both client-id and topic dimensions. The former is an aggregate across all topics of the latter.

// GAUGE(kafka.consumer.records-consumed-rate): Average number of records consumed per second. This metric has either
// client-id dimension or, both client-id and topic dimensions. The former is an aggregate across all topics of the latter.

// GAUGE(kafka.consumer.fetch-rate): Number of records consumed per second.

// GAUGE(kafka.consumer.fetch-size-avg): Average number of bytes fetched per request. This metric has either
// client-id dimension or, both client-id and topic dimensions. The former is an aggregate across all topics of the latter.


var serviceName = "kafka_consumer"

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
