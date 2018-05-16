// +build !windows

package genericjmx

import (
	"github.com/signalfx/signalfx-agent/internal/monitors"
	yaml "gopkg.in/yaml.v2"
)

const monitorType = "collectd/genericjmx"

// MONITOR(collectd/genericjmx): Monitors Java services that expose metrics on
// JMX using collectd's GenericJMX plugin.
//
// See https://github.com/signalfx/integrations/tree/master/collectd-genericjmx
// and https://collectd.org/documentation/manpages/collectd-java.5.shtml
//
// Example (gets the thread count from a standard JMX MBean available on all
// Java JMX-enabled apps):
//
// ```yaml
//
// monitors:
//  - type: collectd/genericjmx
//    host: my-java-app
//    port: 7099
//    mBeanDefinitions:
//      threading:
//        objectName: java.lang:type=Threading
//        values:
//        - type: gauge
//          table: false
//          instancePrefix: jvm.threads.count
//          attribute: ThreadCount
// ```

// Monitor is the main type that represents the monitor
type Monitor struct {
	*JMXMonitorCore
}

func init() {
	err := yaml.Unmarshal([]byte(defaultMBeanYAML), &DefaultMBeans)
	if err != nil {
		panic("YAML for GenericJMX MBeans is invalid: " + err.Error())
	}

	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			NewJMXMonitorCore(DefaultMBeans, "java"),
		}
	}, &Config{})
}
