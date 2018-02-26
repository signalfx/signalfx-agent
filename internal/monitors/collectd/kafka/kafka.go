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
// See https://github.com/signalfx/integrations/tree/master/collectd-kafka.

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
