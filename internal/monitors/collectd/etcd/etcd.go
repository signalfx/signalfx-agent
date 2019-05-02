package etcd

import (
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"

	"github.com/signalfx/signalfx-agent/internal/core/config"

	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
)

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
	// An arbitrary name of the etcd cluster to make it easier to group
	// together and identify instances.
	ClusterName       string `yaml:"clusterName" validate:"required"`
	SSLKeyFile        string `yaml:"sslKeyFile"`
	SSLCertificate    string `yaml:"sslCertificate"`
	SSLCACerts        string `yaml:"sslCACerts"`
	SkipSSLValidation *bool  `yaml:"skipSSLValidation"`
	EnhancedMetrics   *bool  `yaml:"enhancedMetrics"`
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
		ModuleName:    "etcd_plugin",
		ModulePaths:   []string{collectd.MakePythonPluginPath("etcd")},
		TypesDBPaths:  []string{collectd.DefaultTypesDBPath()},
		PluginConfig: map[string]interface{}{
			"Host":                conf.Host,
			"Port":                conf.Port,
			"Interval":            conf.IntervalSeconds,
			"Cluster":             conf.ClusterName,
			"ssl_cert_validation": conf.SkipSSLValidation,
			"EnhancedMetrics":     conf.EnhancedMetrics,
			"ssl_keyfile":         conf.SSLKeyFile,
			"ssl_certificate":     conf.SSLCertificate,
			"ssl_ca_certs":        conf.SSLCACerts,
		},
	}

	return m.PyMonitor.Configure(conf)
}
