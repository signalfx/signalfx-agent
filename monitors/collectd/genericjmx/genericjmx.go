// Package genericjmx coordinates the various monitors that rely on the
// GenericJMX Collectd plugin to pull JMX metrics.  All of the GenericJMX
// monitors share the same instance of the coreInstance struct that can be
// gotten by the Instance func in this package.  This ultimately means that all
// GenericJMX config will be written to one file to make it simpler to control
// dependencies.
package genericjmx

import (
	"sync"

	yaml "gopkg.in/yaml.v2"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/monitors/collectd"
)

//go:generate collectd-template-to-go genericjmx.tmpl

type connection struct {
	ServiceName     string
	MBeansToCollect []string
}

// Config has configuration that is specific to GenericJMX. This config should
// be used by a monitors that use the generic JMX collectd plugin.
type Config struct {
	config.MonitorConfig
	ServiceName      *string                 `yaml:"serviceName"`
	ServiceURL       *string                 `yaml:"serviceURL"`
	InstancePrefix   *string                 `yaml:"instancePrefix"`
	Username         *string                 `yaml:"username"`
	Password         *string                 `yaml:"password"`
	MBeansToCollect  []string                `yaml:"mBeansToCollect"`
	MBeanDefinitions MBeanMap                `yaml:"mBeanDefinitions"`
	ServiceEndpoints []services.EndpointCore `yaml:"serviceEndpoints" default:"[]"`
}

// MonitorCore should be embedded by all monitors that use the
// collectd GenericJMX plugin.  It has most of the logic they will need.  The
// individual monitors mainly just need to provide their set of default mBean
// definitions.
type MonitorCore struct {
	collectd.ServiceMonitorCore

	defaultMBeans      MBeanMap
	defaultServiceName string
	lock               sync.Mutex
}

func NewMonitorCore(defaultMBeans MBeanMap, defaultServiceName string) *MonitorCore {
	mc := &MonitorCore{
		ServiceMonitorCore: *collectd.NewServiceMonitorCore(CollectdTemplate),
		defaultMBeans:      defaultMBeans,
		defaultServiceName: defaultServiceName,
	}

	// This is pretty hacky reaching through two intermediate objects but it
	// would be very verbose trying to propagate it through
	// constructors/setters.
	mc.ServiceMonitorCore.BaseMonitor.UsesGenericJMX = true
	return mc
}

// Configure configures and runs the plugin in collectd
func (m *MonitorCore) Configure(conf *Config) bool {
	if conf.MBeanDefinitions == nil {
		conf.MBeanDefinitions = m.defaultMBeans
	}
	if conf.MBeansToCollect == nil {
		conf.MBeansToCollect = conf.MBeanDefinitions.MBeanNames()
	}
	if conf.ServiceName == nil {
		conf.ServiceName = &m.defaultServiceName
	}

	return m.SetConfigurationAndRun(conf)
}

func init() {
	err := yaml.Unmarshal([]byte(defaultMBeanYAML), &DefaultMBeans)
	if err != nil {
		panic("YAML for GenericJMX MBeans is invalid: " + err.Error())
	}
}
