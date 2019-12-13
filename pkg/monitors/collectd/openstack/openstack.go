package openstack

import (
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors/collectd"

	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/pkg/monitors/subproc"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} {
		return &Monitor{
			python.PyMonitor{
				MonitorCore: subproc.New(),
			},
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"false"`
	python.CommonConfig  `yaml:",inline"`
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
	c.pyConf.CommonConfig = c.CommonConfig
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
		ModulePaths:   []string{collectd.MakePythonPluginPath("openstack")},
		TypesDBPaths:  []string{collectd.DefaultTypesDBPath()},
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
