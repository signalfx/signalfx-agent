// Package genericjmx coordinates the various monitors that rely on the
// GenericJMX Collectd plugin to pull JMX metrics.
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

type serviceEndpoint struct {
	services.EndpointCore `yaml:",inline"`
	ServiceName           *string   `yaml:"serviceName"`
	ServiceURL            *string   `yaml:"serviceURL"`
	InstancePrefix        *string   `yaml:"instancePrefix"`
	Username              *string   `yaml:"username"`
	Password              *string   `yaml:"password"`
	MBeansToCollect       *[]string `yaml:"mBeansToCollect"`
	MBeanDefinitions      MBeanMap  `yaml:"mBeanDefinitions"`
}

type connection struct {
	ServiceName     string
	MBeansToCollect []string
}

// Config has configuration that is specific to GenericJMX. This config should
// be used by a monitors that use the generic JMX collectd plugin.
type Config struct {
	config.MonitorConfig
	Common           serviceEndpoint   `yaml:",inline" default:"{}"`
	ServiceEndpoints []serviceEndpoint `yaml:"serviceEndpoints" default:"[]"`
}

// MonitorCore should be embedded by all monitors that use the
// collectd GenericJMX plugin.  It has most of the logic they will need.  The
// individual monitors mainly just need to provide their set of default mBean
// definitions.
type MonitorCore struct {
	collectd.ServiceMonitorCore

	configs           map[types.MonitorID]*Config
	configForEndpoint map[services.ID]*Config
	lock              sync.Mutex
}

var coreInstance MonitorCore

func init() {
	err := yaml.Unmarshal([]byte(defaultMBeanYAML), &defaultMBeans)
	if err != nil {
		panic("YAML for GenericJMX MBeans is invalid: " + err.Error())
	}

	coreInstance = MonitorCore{
		ServiceMonitorCore: *collectd.NewServiceMonitorCore(CollectdTemplate),
		configs:            make(map[types.MonitorID]*Config),
		configForEndpoint:  make(map[services.ID]*Config),
	}
	coreInstance.Context["mBeanDefinitions"] = defaultMBeans
	coreInstance.Context["configForEndpoint"] = coreInstance.configForEndpoint
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

	gmc.Context["mBeanDefinitions"] = gmc.Context["mBeanDefinitions"].(MBeanMap).MergeWith(conf.Common.MBeanDefinitions)

	var allMBeanNames []string
	for k := range defaultMBeans.MergeWith(conf.Common.MBeanDefinitions) {
		allMBeanNames = append(allMBeanNames, k)
	}

	if conf.Common.MBeansToCollect == nil || len(*conf.Common.MBeansToCollect) == 0 {
		conf.Common.MBeansToCollect = &allMBeanNames
	}

	gmc.configs[conf.ID] = conf

	gmc.SetConfiguration(&conf.MonitorConfig)

	gmc.WriteConfigForPluginAndRestart()
}

// RemoveConfiguration should be called by individual monitors when they are
// shutdown.
func (gmc *MonitorCore) RemoveConfiguration(conf *Config) {
	gmc.lock.Lock()
	defer gmc.lock.Unlock()

	delete(gmc.configs, conf.ID)
	// Don't remove MBeans for removed configuration because it is somewhat
	// complex to do properly in the face of multiple monitors with overlapping
	// MBeans.  It won't hurt anything to have them in the plugin config since
	// they won't be collected if they aren't specified in a service endpoint.
}

// AddService is called by the monitor manager when services are added. It will
// rewrite the shared config file and queue a collectd restart.
func (gmc *MonitorCore) AddService(service services.Endpoint) {
	gmc.lock.Lock()
	defer gmc.lock.Unlock()

	var serviceMatchesConfig bool
	for id, config := range gmc.configs {
		serviceMatchesConfig = service.MatchingMonitors()[id]
		if serviceMatchesConfig {
			log.Debugf("Service matches config: %s\n%s", spew.Sdump(service), config)
			gmc.configForEndpoint[service.ID()] = config

			for dim, value := range config.ExtraDimensions {
				service.AddDimension(dim, value)
			}
			break
		}
	}
	if !serviceMatchesConfig {
		log.WithFields(log.Fields{
			"serviceMatchingMonitors": spew.Sdump(service.MatchingMonitors()),
			"configs":                 spew.Sdump(gmc.configs),
		}).Error("Service did not have any matching GenericJMX monitors!")
	}

	gmc.ServiceMonitorCore.AddService(service)
}

// RemoveService undoes what AddService does.
func (gmc *MonitorCore) RemoveService(service services.Endpoint) {
	gmc.lock.Lock()
	defer gmc.lock.Unlock()

	delete(gmc.configForEndpoint, service.ID())
	gmc.ServiceMonitorCore.RemoveService(service)
}
