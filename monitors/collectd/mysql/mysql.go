package mysql

//go:generate collectd-template-to-go mysql.tmpl

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
	log "github.com/sirupsen/logrus"
)

const monitorType = "collectd/mysql"

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
	Databases []struct {
		Name     string  `yaml:"name"`
		Username string  `yaml:"username"`
		Password *string `yaml:"password"`
	} `yaml:"databases" required:"true"`
	// These credentials serve as defaults for all databases if not overridden
	Username         string                  `yaml:"username"`
	Password         *string                 `yaml:"password"`
	ReportHost       bool                    `yaml:"reportHost" default:"false"`
	ServiceEndpoints []services.EndpointCore `yaml:"serviceEndpoints" default:"[]"`
}

func (c *Config) Validate() bool {
	if len(c.Databases) == 0 {
		log.Error("You must specify at least one database for MySQL")
		return false
	}

	for _, db := range c.Databases {
		if db.Username == "" && c.Username == "" {
			log.Error("Username is required for MySQL monitoring")
			return false
		}
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
