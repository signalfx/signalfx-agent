// Package genericjmx coordinates the various monitors that rely on the
// GenericJMX Collectd plugin to pull JMX metrics.  All of the GenericJMX
// monitors share the same instance of the coreInstance struct that can be
// gotten by the Instance func in this package.  This ultimately means that all
// GenericJMX config will be written to one file to make it simpler to control
// dependencies.
package genericjmx

import (
	"sync"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

//go:generate collectd-template-to-go genericjmx.tmpl

type connection struct {
	ServiceName     string
	MBeansToCollect []string
}

// Config has configuration that is specific to GenericJMX. This config should
// be used by a monitors that use the generic JMX collectd plugin.
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`

	Host string  `yaml:"host" validate:"required"`
	Port uint16  `yaml:"port" validate:"required"`
	Name *string `yaml:"name"`

	ServiceName      string   `yaml:"serviceName"`
	ServiceURL       string   `yaml:"serviceURL"`
	InstancePrefix   string   `yaml:"instancePrefix"`
	Username         string   `yaml:"username"`
	Password         string   `yaml:"password"`
	MBeansToCollect  []string `yaml:"mBeansToCollect"`
	MBeanDefinitions MBeanMap `yaml:"mBeanDefinitions"`
}

// JMXMonitorCore should be embedded by all monitors that use the collectd
// GenericJMX plugin.  It has most of the logic they will need.  The individual
// monitors mainly just need to provide their set of default mBean definitions.
type JMXMonitorCore struct {
	collectd.MonitorCore

	defaultMBeans      MBeanMap
	defaultServiceName string
	lock               sync.Mutex
}

// NewJMXMonitorCore makes a new JMX core as well as the underlying MonitorCore
func NewJMXMonitorCore(defaultMBeans MBeanMap, defaultServiceName string) *JMXMonitorCore {
	mc := &JMXMonitorCore{
		MonitorCore:        *collectd.NewMonitorCore(CollectdTemplate),
		defaultMBeans:      defaultMBeans,
		defaultServiceName: defaultServiceName,
	}

	mc.MonitorCore.UsesGenericJMX = true
	return mc
}

// Configure configures and runs the plugin in collectd
func (m *JMXMonitorCore) Configure(conf *Config) error {
	conf.MBeanDefinitions = m.defaultMBeans.MergeWith(conf.MBeanDefinitions)
	if conf.MBeansToCollect == nil {
		conf.MBeansToCollect = conf.MBeanDefinitions.MBeanNames()
	}
	if conf.ServiceName == "" {
		conf.ServiceName = m.defaultServiceName
	}

	return m.SetConfigurationAndRun(conf)
}
