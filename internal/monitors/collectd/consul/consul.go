// +build !windows

package consul

//go:generate collectd-template-to-go consul.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
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
			*collectd.NewMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`

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
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) error {
	return am.SetConfigurationAndRun(conf)
}
