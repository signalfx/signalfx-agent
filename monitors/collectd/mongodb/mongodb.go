package mongodb

//go:generate collectd-template-to-go mongodb.tmpl

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
	log "github.com/sirupsen/logrus"
)

const monitorType = "collectd/mongodb"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			*collectd.NewServiceMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig
	Databases              []string                `yaml:"databases"`
	Username               string                  `yaml:"username"`
	Password               *string                 `yaml:"password"`
	UseTLS                 *bool                   `yaml:"useTLS"`
	CACerts                *string                 `yaml:"caCerts"`
	TLSClientCert          *string                 `yaml:"tlsClientCert"`
	TLSClientKey           *string                 `yaml:"tlsClientKey"`
	TLSClientKeyPassPhrase *string                 `yaml:"tlsClientKeyPassPhrase"`
	ServiceEndpoints       []services.EndpointCore `yaml:"serviceEndpoints" default:"[]"`
}

func (c *Config) Validate() bool {
	if len(c.Databases) == 0 {
		log.Error("You must specify at least one database for MongoDB")
		return false
	}
	if c.Username == "" {
		log.Error("You must specify a username for MongoDB")
		return false
	}
	return true
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.ServiceMonitorCore
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) bool {
	return am.SetConfigurationAndRun(conf)
}
