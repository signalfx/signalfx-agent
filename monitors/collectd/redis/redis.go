package redis

//go:generate collectd-template-to-go redis.tmpl

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
)

const monitorType = "collectd/redis"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &RedisMonitor{
			*collectd.NewServiceMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

type Config struct {
	config.MonitorConfig
	Auth string
}

type RedisMonitor struct {
	collectd.ServiceMonitorCore
}

func (rm *RedisMonitor) Configure(conf *Config) bool {
	rm.Context["auth"] = conf.Auth
	return rm.SetConfigurationAndRun(conf.MonitorConfig)
}
