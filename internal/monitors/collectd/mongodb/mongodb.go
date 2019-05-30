package mongodb

import (
	"errors"

	"github.com/signalfx/golib/pointer"
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
	config.MonitorConfig     `yaml:",inline" acceptsEndpoints:"true"`
	pyConf                   *python.Config
	Host                     string   `yaml:"host" validate:"required"`
	Port                     uint16   `yaml:"port" validate:"required"`
	Databases                []string `yaml:"databases" validate:"required"`
	Username                 string   `yaml:"username"`
	Password                 string   `yaml:"password" neverLog:"true"`
	UseTLS                   *bool    `yaml:"useTLS"`
	CACerts                  string   `yaml:"caCerts"`
	TLSClientCert            string   `yaml:"tlsClientCert"`
	TLSClientKey             string   `yaml:"tlsClientKey"`
	TLSClientKeyPassPhrase   string   `yaml:"tlsClientKeyPassPhrase"`
	SendCollectionMetrics    *bool    `yaml:"sendCollectionMetrics"`
	SendCollectionTopMetrics *bool    `yaml:"sendCollectionTopMetrics"`
}

// PythonConfig returns the embedded python.Config struct from the interface
func (c *Config) PythonConfig() *python.Config {
	return c.pyConf
}

// Validate will check the config for correctness.
func (c *Config) Validate() error {
	if len(c.Databases) == 0 {
		return errors.New("must specify at least one database for MongoDB")
	}
	return nil
}

// GetExtraMetrics returns a list of metrics that should be let through the
// filtering based on config flags.
func (c *Config) GetExtraMetrics() []string {
	var out []string
	if c.SendCollectionMetrics != nil && *c.SendCollectionMetrics {
		out = append(out, groupMetricsMap[groupCollection]...)
	}
	if c.SendCollectionTopMetrics != nil && *c.SendCollectionTopMetrics {
		out = append(out, groupMetricsMap[groupCollectionTop]...)
	}
	return out
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.PyMonitor
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	sendCollMetrics := conf.SendCollectionMetrics
	sendCollTopMetrics := conf.SendCollectionTopMetrics

	if m.Output.HasEnabledMetricInGroup(groupCollection) {
		sendCollMetrics = pointer.Bool(true)
	}
	if m.Output.HasEnabledMetricInGroup(groupCollectionTop) {
		sendCollTopMetrics = pointer.Bool(true)
	}

	conf.pyConf = &python.Config{
		MonitorConfig: conf.MonitorConfig,
		Host:          conf.Host,
		Port:          conf.Port,
		ModuleName:    "mongodb",
		ModulePaths:   []string{collectd.MakePythonPluginPath("mongodb")},
		TypesDBPaths:  []string{collectd.DefaultTypesDBPath()},
		PluginConfig: map[string]interface{}{
			"Host":                     conf.Host,
			"Port":                     conf.Port,
			"Database":                 conf.Databases,
			"UseTLS":                   conf.UseTLS,
			"User":                     conf.Username,
			"Password":                 conf.Password,
			"CACerts":                  conf.CACerts,
			"TLSClientCert":            conf.TLSClientCert,
			"TLSClientKey":             conf.TLSClientKey,
			"TLSClientKeyPassphrase":   conf.TLSClientKeyPassPhrase,
			"SendCollectionMetrics":    sendCollMetrics,
			"SendCollectionTopMetrics": sendCollTopMetrics,
		},
	}

	return m.PyMonitor.Configure(conf)
}
