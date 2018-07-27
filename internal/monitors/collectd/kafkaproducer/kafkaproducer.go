// +build !windows

package kafkaproducer

import (
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/genericjmx"
	yaml "gopkg.in/yaml.v2"
)

const monitorType = "collectd/kafka_producer"

// MONITOR(collectd/kafka_producer): Monitors a java based Kafka producer using GenericJMX.
//
// See the [integration documentation](https://github.com/signalfx/integrations/tree/master/collectd-kafka_producer)
// for more information.
//
// This monitor has a set of [built in MBeans
// configured](https://github.com/signalfx/signalfx-agent/tree/master/internal/monitors/collectd/kafka_producer/mbeans.go)
// for which it pulls metrics from the Kafka producer's JMX endpoint.
//
// Sample YAML configuration:
//```yaml
// monitors:
//   - type: collectd/kafka_producer
//     host: localhost
//     port: 8099
// ```

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
