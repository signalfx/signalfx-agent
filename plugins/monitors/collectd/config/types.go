package config

import (
	"fmt"

	"github.com/signalfx/neo-agent/services"
)

// Plugin describes a collectd plugin
type Plugin struct {
	Name      string
	Dims      string
	Host      string
	Port      uint16
	Templates []string
	Config    map[string]interface{}
}

// PLUGINS is a mapping to create plugin instances with defaults
var PLUGINS = map[services.ServiceType]func(string) *Plugin{
	services.ApacheService: func(pluginName string) *Plugin {
		return &Plugin{
			Templates: []string{"apache.conf.tmpl"},
			Name:      pluginName,
			Host:      "localhost",
			Port:      80}
	},
	services.DockerService: func(pluginName string) *Plugin {
		return &Plugin{
			Templates: []string{"docker.conf.tmpl"},
			Name:      pluginName,
			Config: map[string]interface{}{
				"hostUrl": "unix:///var/run/docker.sock",
			},
		}
	},
	services.MongoDBService: func(pluginName string) *Plugin {
		return &Plugin{
			Templates: []string{"mongodb.conf.tmpl"},
			Name:      pluginName,
			Host:      "localhost",
			Port:      27017}
	},
	services.RedisService: func(pluginName string) *Plugin {
		return &Plugin{
			Templates: []string{"redis-master.conf.tmpl"},
			Name:      pluginName,
			Host:      "localhost",
			Port:      6379,
		}
	},
	services.SignalfxService: func(pluginName string) *Plugin {
		return &Plugin{
			Templates: []string{
				"signalfx.conf.tmpl",
				"write-http.conf.tmpl"},
			Config: map[string]interface{}{
				"ingestUrl": "https://ingest.signalfx.com",
			},
			Name: pluginName,
		}
	},
	services.ZookeeperService: func(pluginName string) *Plugin {
		return &Plugin{
			Templates: []string{"zookeeper.conf.tmpl"},
			Name:      pluginName,
			Host:      "localhost",
			Port:      2181}
	},
}

// NewPlugin constructs a plugin with default values depending on the service type
func NewPlugin(pluginType services.ServiceType, pluginName string) (*Plugin, error) {
	if create, ok := PLUGINS[pluginType]; ok {
		return create(pluginName), nil
	}
	return nil, fmt.Errorf("plugin %s is unsupported", pluginType)
}

// CollectdConfig are global collectd settings
type CollectdConfig struct {
	Interval             uint
	Timeout              uint
	ReadThreads          uint
	WriteQueueLimitHigh  uint `yaml:"writeQueueLimitHigh"`
	WriteQueueLimitLow   uint `yaml:"writeQueueLimitLow"`
	CollectInternalStats bool
	Plugins              []map[string]interface{}
}

// AppConfig is the top-level configuration object consumed by templates.
type AppConfig struct {
	AgentConfig *CollectdConfig
	Plugins     []*Plugin
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
