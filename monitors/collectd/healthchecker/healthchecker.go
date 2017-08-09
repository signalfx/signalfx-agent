package healthchecker

//go:generate collectd-template-to-go healthchecker.tmpl

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
)

const monitorType = "collectd/health_checker"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &HealthCheckerMonitor{
			*collectd.NewServiceMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

type Config struct {
	config.MonitorConfig
	URL string
	// This can be either a string or numberic type
	JSONVal interface{}
	JSONKey string
}

type HealthCheckerMonitor struct {
	collectd.ServiceMonitorCore
}

func (rm *HealthCheckerMonitor) Configure(conf *Config) bool {
	rm.Context["URL"] = conf.URL
	rm.Context["JSONKey"] = conf.JSONKey
	rm.Context["JSONVal"] = conf.JSONVal
	return rm.SetConfigurationAndRun(conf.MonitorConfig)
}
