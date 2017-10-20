package genericjmx

import (
	"github.com/signalfx/neo-agent/monitors"
)

const monitorType = "collectd/genericjmx"

// Monitor is the main type that represents the monitor
type Monitor struct {
	*MonitorCore
}

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			Instance(),
		}
	}, &Config{})
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) bool {
	conf.MBeanDefinitions = conf.MBeanDefinitions.MergeWith(DefaultMBeans)
	m.AddConfiguration(conf)
	return true
}
