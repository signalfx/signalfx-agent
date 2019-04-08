package consul

import (
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"

	"github.com/signalfx/signalfx-agent/internal/core/config"

	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
)

const monitorType = "collectd/consul"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			python.PyMonitor{
				MonitorCore: pyrunner.New("sfxcollectd"),
			},
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	pyConf               *python.Config
	Host                 string `yaml:"host" validate:"required"`
	Port                 uint16 `yaml:"port" validate:"required"`
	ACLToken             string `yaml:"aclToken" neverLog:"true"`
	UseHTTPS             bool   `yaml:"useHTTPS"`
	EnhancedMetrics      *bool  `yaml:"enhancedMetrics"`
	CACertificate        string `yaml:"caCertificate"`
	ClientCertificate    string `yaml:"clientCertificate"`
	ClientKey            string `yaml:"clientKey"`
	SignalFxAccessToken  string `yaml:"signalFxAccessToken" neverLog:"true"`
}

// PythonConfig returns the embedded python.Config struct from the interface
func (c *Config) PythonConfig() *python.Config {
	return c.pyConf
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.PyMonitor
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	conf.pyConf = &python.Config{
		MonitorConfig: conf.MonitorConfig,
		Host:          conf.Host,
		Port:          conf.Port,
		ModuleName:    "consul_plugin",
		ModulePaths:   []string{collectd.MakePath("consul")},
		TypesDBPaths:  []string{collectd.MakePath("types.db")},
		PluginConfig: map[string]interface{}{
			"ApiHost":           conf.Host,
			"ApiPort":           conf.Port,
			"TelemetryServer":   false,
			"SfxToken":          conf.SignalFxAccessToken,
			"EnhancedMetrics":   conf.EnhancedMetrics,
			"AclToken":          conf.ACLToken,
			"CaCertificate":     conf.CACertificate,
			"ClientCertificate": conf.ClientCertificate,
			"ClientKey":         conf.ClientKey,
		},
	}

	if conf.UseHTTPS {
		conf.pyConf.PluginConfig["ApiProtocol"] = "https"
	} else {
		conf.pyConf.PluginConfig["ApiProtocol"] = "http"
	}

	return m.PyMonitor.Configure(conf)
}
