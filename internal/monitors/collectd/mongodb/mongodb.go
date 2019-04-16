package mongodb

import (
	"errors"

	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"

	"github.com/signalfx/signalfx-agent/internal/core/config"

	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"

	"github.com/signalfx/signalfx-agent/internal/monitors"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} {
		return &Monitor{
			python.PyMonitor{
				MonitorCore: pyrunner.New("sfxcollectd"),
			},
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig   `yaml:",inline" acceptsEndpoints:"true"`
	pyConf                 *python.Config
	Host                   string   `yaml:"host" validate:"required"`
	Port                   uint16   `yaml:"port" validate:"required"`
	Databases              []string `yaml:"databases" validate:"required"`
	Username               string   `yaml:"username"`
	Password               string   `yaml:"password" neverLog:"true"`
	UseTLS                 *bool    `yaml:"useTLS"`
	CACerts                string   `yaml:"caCerts"`
	TLSClientCert          string   `yaml:"tlsClientCert"`
	TLSClientKey           string   `yaml:"tlsClientKey"`
	TLSClientKeyPassPhrase string   `yaml:"tlsClientKeyPassPhrase"`
}

// PythonConfig returns the embedded python.Config struct from the interface
func (c *Config) PythonConfig() *python.Config {
	return c.pyConf
}

// Validate will check the config for correctness.
func (c *Config) Validate() error {
	if len(c.Databases) == 0 {
		return errors.New("You must specify at least one database for MongoDB")
	}
	return nil
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
		ModuleName:    "mongodb",
		ModulePaths:   []string{collectd.MakePath("mongodb")},
		TypesDBPaths:  []string{collectd.MakePath("types.db")},
		PluginConfig: map[string]interface{}{
			"Host":                   conf.Host,
			"Port":                   conf.Port,
			"Database":               conf.Databases,
			"UseTLS":                 conf.UseTLS,
			"User":                   conf.Username,
			"Password":               conf.Password,
			"CACerts":                conf.CACerts,
			"TLSClientCert":          conf.TLSClientCert,
			"TLSClientKey":           conf.TLSClientKey,
			"TLSClientKeyPassphrase": conf.TLSClientKeyPassPhrase,
		},
	}

	return m.PyMonitor.Configure(conf)
}
