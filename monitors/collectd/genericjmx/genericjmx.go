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

	"github.com/davecgh/go-spew/spew"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/config/types"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/monitors/collectd"
	log "github.com/sirupsen/logrus"
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

// A custom config struct that holds each of the config structs for all of the
// genericJMX-based monitors.
type mergedConfig struct {
	*config.MonitorConfig
	ByMonitorID  map[types.MonitorID]*Config
	ByEndpointID map[services.ID]*Config
}

// MonitorCore should be embedded by all monitors that use the
// collectd GenericJMX plugin.  It has most of the logic they will need.  The
// individual monitors mainly just need to provide their set of default mBean
// definitions.
type MonitorCore struct {
	collectd.ServiceMonitorCore

	config mergedConfig
	lock   sync.Mutex
}

var coreInstance MonitorCore

func init() {
	err := yaml.Unmarshal([]byte(defaultMBeanYAML), &DefaultMBeans)
	if err != nil {
		panic("YAML for GenericJMX MBeans is invalid: " + err.Error())
	}

	coreInstance = MonitorCore{
		ServiceMonitorCore: *collectd.NewServiceMonitorCore(CollectdTemplate),
		config: mergedConfig{
			MonitorConfig: &config.MonitorConfig{},
			ByMonitorID:   make(map[types.MonitorID]*Config),
			ByEndpointID:  make(map[services.ID]*Config),
		},
	}
}

// Instance returns the shared monitor core instance.
func Instance() *MonitorCore {
	return &coreInstance
}

// AddConfiguration should be called by the individual monitors with the config
// that they have received in Configure.
func (gmc *MonitorCore) AddConfiguration(conf *Config) {
	gmc.lock.Lock()
	defer gmc.lock.Unlock()

	gmc.config.ByMonitorID[conf.ID] = conf

	// In order to configure Collectd, we have to have a single config for all
	// of the GenericJMX configurations.  Since the Collectd config is
	// propagated down to all monitor config structs, we can just assign each
	// monitor's core config to this and override it each time.
	*gmc.config.MonitorConfig = *conf.CoreConfig()

	// We do need to override the ID though so that it stays the same so
	// Collectd can track this in case it needs to be shutdown.
	gmc.config.MonitorConfig.ID = "genericjmx"
	gmc.config.MonitorConfig.ExtraDimensions = nil

	if len(conf.MBeansToCollect) == 0 {
		conf.MBeansToCollect = conf.MBeanDefinitions.MBeanNames()
	}
	gmc.SetConfiguration(gmc.config)

	gmc.WriteConfigForPluginAndRestart()
}

// RemoveConfiguration should be called by individual monitors when they are
// shutdown.
func (gmc *MonitorCore) RemoveConfiguration(conf *Config) {
	gmc.lock.Lock()
	defer gmc.lock.Unlock()

	delete(gmc.config.ByMonitorID, conf.ID)
}

// AddService is called by the monitor manager when services are added. It will
// rewrite the shared config file and queue a collectd restart.
func (gmc *MonitorCore) AddService(service services.Endpoint) {
	gmc.lock.Lock()
	defer gmc.lock.Unlock()

	var serviceMatchesConfig bool
	for id, config := range gmc.config.ByMonitorID {
		serviceMatchesConfig = service.MatchingMonitors()[id]
		if serviceMatchesConfig {
			log.Debugf("Service matches config: %s\n%s", spew.Sdump(service), config)
			gmc.config.ByEndpointID[service.ID()] = config
			break
		}
	}
	if !serviceMatchesConfig {
		log.WithFields(log.Fields{
			"serviceMatchingMonitors": spew.Sdump(service.MatchingMonitors()),
			"config":                  spew.Sdump(gmc.config.ByMonitorID),
		}).Error("Service did not have any matching GenericJMX monitors!")
	}

	gmc.ServiceMonitorCore.AddService(service)
}

// RemoveService undoes what AddService does.
func (gmc *MonitorCore) RemoveService(service services.Endpoint) {
	gmc.lock.Lock()
	defer gmc.lock.Unlock()

	delete(gmc.config.ByEndpointID, service.ID())
	gmc.ServiceMonitorCore.RemoveService(service)
}
