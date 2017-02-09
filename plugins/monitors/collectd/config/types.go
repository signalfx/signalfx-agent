package config

import (
	"fmt"

	"github.com/signalfx/neo-agent/services"
)

type Plugin struct {
	Templates []string
	Name      string
}

type IPlugin interface {
	// Normally in Go this would just be `Templates()` but that would
	// collide with the field `Templates` which has to be public for
	// YAML's sake.
	GetTemplates() []string
}

// IHostPort can be used for plugins that take a host and port number
type IHostPort interface {
	SetHost(host string)
	SetPort(port uint16)
}

func (plugin *Plugin) GetTemplates() []string {
	return plugin.Templates
}

// NewPlugin constructs a plugin-specific instance of IPlugin.
func NewPlugin(pluginType services.ServiceType, pluginName string) (IPlugin, error) {
	switch pluginType {
	case services.SIGNALFX:
		return NewSignalFxConfig(pluginName), nil
	case services.REDIS:
		return NewRedisConfig(pluginName), nil
	case services.APACHE:
		return NewApacheConfig(pluginName), nil
	default:
		return nil, fmt.Errorf("plugin %s is unsupported", pluginType)
	}
}

// SetHost sets hostname
func (hp *HostPort) SetHost(host string) {
	hp.Host = host
}

// SetPort sets port number
func (hp *HostPort) SetPort(port uint16) {
	hp.Port = port
}

type CollectdConfig struct {
	Interval             uint
	Timeout              uint
	ReadThreads          uint
	WriteQueueLimitHigh  uint `yaml:"writeQueueLimitHigh"`
	WriteQueueLimitLow   uint `yaml:"writeQueueLimitLow"`
	CollectInternalStats bool
	Plugins              []map[string]interface{}
}

type HostPort struct {
	Host string
	Port uint16
}

type RedisConfig struct {
	Plugin   `yaml:",inline"`
	HostPort `yaml:",inline"`
}

type ApacheConfig struct {
	Plugin   `yaml:",inline"`
	HostPort `yaml:",inline"`
}

type SignalFxConfig struct {
	Plugin          `yaml:",inline"`
	IngestUrl       string `yaml:"ingestUrl"`
	ApiToken        string `yaml:"apiToken"`
	ExtraDimensions string `yaml:"extraDimensions`
}

// AppConfig is the top-level configuration object consumed by templates.
type AppConfig struct {
	AgentConfig *CollectdConfig
	Plugins     []IPlugin
}

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

func NewRedisConfig(pluginName string) *RedisConfig {
	return &RedisConfig{
		Plugin{
			Templates: []string{"redis-master.conf.tmpl"},
			Name:      pluginName},
		HostPort{
			Host: "localhost",
			Port: 6379},
	}
}

func NewSignalFxConfig(pluginName string) *SignalFxConfig {
	return &SignalFxConfig{
		IngestUrl: "https://ingest.signalfx.com",
		Plugin: Plugin{
			Templates: []string{"signalfx.conf.tmpl", "write-http.conf.tmpl"},
			Name:      pluginName},
	}
}

func NewApacheConfig(pluginName string) *ApacheConfig {
	return &ApacheConfig{
		Plugin{
			Templates: []string{"apache.conf.tmpl"},
			Name:      pluginName},
		HostPort{
			Host: "localhost",
			Port: 80},
	}
}
