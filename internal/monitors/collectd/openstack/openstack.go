package openstack

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"

	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
)

const monitorType = "collectd/openstack"

// MONITOR(collectd/openstack): Monitors Openstack by using the
// [Openstack collectd Python
// plugin](https://github.com/signalfx/collectd-openstack), which collects metrics
// from Openstack instances
//
// ```yaml
// monitors:
// - type: collectd/openstack
//   authURL: "http://192.168.11.111/identity/v3"
//   username: "admin"
//   password: "secret"
// ```

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
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"false"`
	pyConf               *python.Config
	// Keystone authentication URL/endpoint for the OpenStack cloud
	AuthURL string `yaml:"authURL" validate:"required"`
	// Username to authenticate with keystone identity
	Username string `yaml:"username" validate:"required"`
	// Password to authenticate with keystone identity
	Password string `yaml:"password" validate:"required"`
	// Specify the name of Project to be monitored (**default**:"demo")
	ProjectName string `yaml:"projectName"`
	// The project domain (**default**:"default")
	ProjectDomainID string `yaml:"projectDomainID"`
	// The user domain id (**default**:"default")
	UserDomainID string `yaml:"userDomainID"`
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
		ModuleName:    "openstack_metrics",
		ModulePaths:   []string{collectd.MakePath("openstack")},
		TypesDBPaths:  []string{collectd.MakePath("types.db")},
		MonitorConfig: conf.MonitorConfig,
		PluginConfig: map[string]interface{}{
			"AuthURL":         conf.AuthURL,
			"Username":        conf.Username,
			"Password":        conf.Password,
			"ProjectName":     conf.ProjectName,
			"ProjectDomainId": conf.ProjectDomainID,
			"UserDomainId":    conf.UserDomainID,
		},
	}

	return m.PyMonitor.Configure(conf)
}
