package config

import (
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/spf13/viper"
)

// Instance describes a collectd plugin
type Instance struct {
	Plugin       string
	Name         string
	Template     string
	TemplateFile string
	Dims         string
	Host         string
	Port         uint16
	Vars         map[string]interface{}
}

// PluginType is a type for collectd plugins
type PluginType string

// Define static collectd plugins
const (
	// SignalFx plugin
	SignalFx PluginType = "signalfx"
	// WriteHTTP plugin
	WriteHTTP PluginType = "writehttp"
	// Docker plugin
	Docker PluginType = "docker"
	// MesosMaster plugin
	MesosMaster PluginType = "mesos-master"
	// MesosAgent plugin
	MesosAgent PluginType = "mesos-agent"
	// Marathon plugin
	Marathon PluginType = "marathon"
)

// PLUGINS is a mapping to create plugin instances with defaults
var PLUGINS = map[PluginType]func(string) *Instance{
	SignalFx: func(instanceName string) *Instance {
		// XXX: Super hacky. Ideally this should have no knowledge of the global
		// viper.
		return &Instance{
			Plugin:       "signalfx",
			TemplateFile: "signalfx.conf.tmpl",
			Vars: map[string]interface{}{
				"url": viper.GetString("ingesturl"),
			},
			Name: instanceName,
		}
	},
	WriteHTTP: func(instanceName string) *Instance {
		// XXX: Super hacky. Ideally this should have no knowledge of the global
		// viper.
		query := url.Values{}

		for k, v := range viper.GetStringMapString("dimensions") {
			query["sfxdim_"+k] = []string{v}
		}

		plugin := &Instance{
			Plugin:       "write-http",
			TemplateFile: "write-http.conf.tmpl",
			Vars: map[string]interface{}{
				"url":        viper.GetString("ingesturl"),
				"dimensions": query.Encode(),
			},
			Name: instanceName,
		}

		return plugin
	},
	Docker: func(instanceName string) *Instance {
		return &Instance{
			Plugin:       "docker",
			TemplateFile: "docker.conf.tmpl",
			Name:         instanceName,
			Vars: map[string]interface{}{
				"url": "unix:///var/run/docker.sock",
			},
		}
	},
	MesosMaster: func(instanceName string) *Instance {
		return &Instance{
			Plugin:       "mesos-master",
			TemplateFile: "mesos-master.conf.tmpl",
			Name:         instanceName,
			Vars: map[string]interface{}{
				"host":         getHostname(),
				"port":         "5050",
				"cluster":      "cluster-0",
				"instance":     fmt.Sprintf("master-%s", getHostname()),
				"systemhealth": "false",
				"verbose":      "false",
			},
		}
	},
	MesosAgent: func(instanceName string) *Instance {
		return &Instance{
			Plugin:       "mesos-agent",
			TemplateFile: "mesos-agent.conf.tmpl",
			Name:         instanceName,
			Vars: map[string]interface{}{
				"host":     getHostname(),
				"port":     "5051",
				"cluster":  "cluster-0",
				"instance": fmt.Sprintf("agent-%s", getHostname()),
				"verbose":  "false",
			},
		}
	},
	Marathon: func(instanceName string) *Instance {

		return &Instance{
			Plugin:       "marathon",
			TemplateFile: "marathon.conf.tmpl",
			Name:         instanceName,
			Vars: map[string]interface{}{
				"host":     getHostname(),
				"port":     "8080",
				"username": "",
				"password": "",
			},
		}
	},
}

// NewPlugin constructs a plugin with default values depending on the service type
func NewPlugin(pluginType PluginType, pluginName string) (*Instance, error) {
	if create, ok := PLUGINS[pluginType]; ok {
		return create(pluginName), nil
	}
	return nil, fmt.Errorf("plugin %s is unsupported", pluginType)
}

// NewInstancePlugin creates a plugin for a supported service type
func NewInstancePlugin(pluginType string, pluginName string) (*Instance, error) {
	// TODO: Maintain a list of supported service types for collectd if not all monitors support the same ones.
	return &Instance{Plugin: pluginType, Name: pluginName, Vars: map[string]interface{}{}}, nil
}

// GroupByPlugin creates a map of instances by plugin
func GroupByPlugin(instances []*Instance) map[string]*Plugin {
	pluginMap := map[string]*Plugin{}
	for _, instance := range instances {
		plugin := instance.Plugin

		if val, ok := pluginMap[plugin]; ok {
			pluginMap[plugin].Instances = append(val.Instances, instance)
		} else {
			pluginMap[plugin] = &Plugin{
				Instances: []*Instance{instance},
				Vars:      map[string]interface{}{},
			}
		}
	}
	return pluginMap
}

// CollectdConfig are global collectd settings
type CollectdConfig struct {
	Interval             uint
	Timeout              uint
	ReadThreads          uint
	WriteQueueLimitHigh  uint `yaml:"writeQueueLimitHigh"`
	WriteQueueLimitLow   uint `yaml:"writeQueueLimitLow"`
	CollectInternalStats bool
	Hostname             string
	Plugins              []map[string]interface{}
}

// AppConfig is the top-level configuration object consumed by templates.
type AppConfig struct {
	AgentConfig *CollectdConfig
	Plugins     map[string]*Plugin //[]*Plugin
}

// Plugin is the top level configuration object for a given plugin.
type Plugin struct {
	Vars      map[string]interface{}
	Instances []*Instance
}

// NewCollectdConfig creates a default collectd config instance
func NewCollectdConfig() *CollectdConfig {
	return &CollectdConfig{
		Interval:             15,
		Timeout:              2,
		ReadThreads:          5,
		WriteQueueLimitHigh:  500000,
		WriteQueueLimitLow:   400000,
		CollectInternalStats: true,
	}
}

// getHostname - returns the hostname or logs and error and returns "localhost"
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Print(err)
		return "localhost"
	}
	return hostname
}
