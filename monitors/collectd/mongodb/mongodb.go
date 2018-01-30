package mongodb

//go:generate collectd-template-to-go mongodb.tmpl

import (
	"errors"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
)

const monitorType = "collectd/mongodb"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			*collectd.NewMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"true"`

	Host                   string   `yaml:"host"`
	Port                   uint16   `yaml:"port"`
	Name                   string   `yaml:"name"`
	Databases              []string `yaml:"databases"`
	Username               string   `yaml:"username"`
	Password               *string  `yaml:"password" neverLog:"true"`
	UseTLS                 *bool    `yaml:"useTLS"`
	CACerts                *string  `yaml:"caCerts"`
	TLSClientCert          *string  `yaml:"tlsClientCert"`
	TLSClientKey           *string  `yaml:"tlsClientKey"`
	TLSClientKeyPassPhrase *string  `yaml:"tlsClientKeyPassPhrase"`
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
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) error {
	return am.SetConfigurationAndRun(conf)
}
