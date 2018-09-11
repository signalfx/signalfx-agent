// +build !windows

package consul

import (
	"os"
	"path/filepath"

	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
)

const monitorType = "collectd/consul"

// MONITOR(collectd/consul): Monitors the Consul data store by using the
// [Consul collectd Python
// plugin](https://github.com/signalfx/collectd-consul), which collects metrics
// from Consul instances by hitting these endpoints:
// - [/agent/self](https://www.consul.io/api/agent.html#read-configuration)
// - [/agent/metrics](https://www.consul.io/api/agent.html#view-metrics)
// - [/catalog/nodes](https://www.consul.io/api/catalog.html#list-nodes)
// - [/catalog/node/:node](https://www.consul.io/api/catalog.html#list-services-for-node)
// - [/status/leader](https://www.consul.io/api/status.html#get-raft-leader)
// - [/status/peers](https://www.consul.io/api/status.html#list-raft-peers)
// - [/coordinate/datacenters](https://www.consul.io/api/coordinate.html#read-wan-coordinates)
// - [/coordinate/nodes](https://www.consul.io/api/coordinate.html#read-lan-coordinates)
// - [/health/state/any](https://www.consul.io/api/health.html#list-checks-in-state)

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			python.Monitor{
				MonitorCore: pyrunner.New("sfxcollectd"),
			},
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	python.CoreConfig `yaml:",inline" acceptsEndpoints:"true"`

	Host string `yaml:"host" validate:"required"`
	Port uint16 `yaml:"port" validate:"required"`

	ACLToken            string `yaml:"aclToken" neverLog:"true"`
	UseHTTPS            bool   `yaml:"useHTTPS"`
	EnhancedMetrics     bool   `yaml:"enhancedMetrics"`
	CACertificate       string `yaml:"caCertificate"`
	ClientCertificate   string `yaml:"clientCertificate"`
	ClientKey           string `yaml:"clientKey"`
	SignalFxAccessToken string `yaml:"signalFxAccessToken" neverLog:"true"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.Monitor
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	conf.PluginConfig = map[string]interface{}{
		"ApiHost":         conf.Host,
		"ApiPort":         conf.Port,
		"TelemetryServer": false,
		"SfxToken":        conf.SignalFxAccessToken,
		"EnhancedMetrics": conf.EnhancedMetrics,
	}
	if conf.ModuleName == "" {
		conf.ModuleName = "consul_plugin"
	}
	if len(conf.ModulePaths) == 0 {
		conf.ModulePaths = []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "consul")}
	}
	if len(conf.TypesDBPaths) == 0 {
		conf.TypesDBPaths = []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "types.db")}
	}
	if conf.UseHTTPS {
		conf.PluginConfig["ApiProtocol"] = "https"
	} else {
		conf.PluginConfig["ApiProtocol"] = "http"
	}
	if conf.ACLToken != "" {
		conf.PluginConfig["AclToken"] = conf.ACLToken
	}
	if conf.CACertificate != "" {
		conf.PluginConfig["CaCertificate"] = conf.CACertificate
	}
	if conf.ClientCertificate != "" {
		conf.PluginConfig["ClientCertificate"] = conf.ClientCertificate
	}
	if conf.ClientKey != "" {
		conf.PluginConfig["ClientKey"] = conf.ClientKey
	}

	return m.Monitor.Configure(conf)
}
