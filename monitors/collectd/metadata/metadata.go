package metadata

//go:generate collectd-template-to-go metadata.tmpl

import (
	"errors"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/meta"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
)

const monitorType = "collectd/signalfx-metadata"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			MonitorCore: *collectd.NewMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `singleInstance:"true"`
	WriteServerURL       string `yaml:"writeServerURL"`
	ProcFSPath           string `yaml:"procFSPath"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
	AgentMeta *meta.AgentMeta
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	if m.AgentMeta.CollectdConf == nil {
		return errors.New("Metadata plugin needs collectd config")
	}

	conf.WriteServerURL = collectd.Instance().WriteServerURL()
	conf.ProcFSPath = m.AgentMeta.ProcFSPath

	return m.SetConfigurationAndRun(conf)
}
