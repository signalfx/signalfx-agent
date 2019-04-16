package spark

import (
	"errors"
	"fmt"

	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"

	"github.com/signalfx/signalfx-agent/internal/core/config"

	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"

	"github.com/signalfx/signalfx-agent/internal/monitors"
)

type sparkClusterType string

const (
	sparkStandalone sparkClusterType = "Standalone"
	sparkMesos                       = "Mesos"
	sparkYarn                        = "Yarn"
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
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	pyConf               *python.Config
	Host                 string `yaml:"host" validate:"required"`
	Port                 uint16 `yaml:"port" validate:"required"`
	// Set to `true` when monitoring a master Spark node
	IsMaster bool `yaml:"isMaster" default:"false"`
	// Should be one of `Standalone` or `Mesos` or `Yarn`.  Cluster metrics will
	// not be collected on Yarn.  Please use the collectd/hadoop monitor to gain
	// insights to your cluster's health.
	ClusterType               sparkClusterType `yaml:"clusterType" validate:"required"`
	CollectApplicationMetrics *bool            `yaml:"collectApplicationMetrics" default:"false"`
	EnhancedMetrics           *bool            `yaml:"enhancedMetrics" default:"false"`
}

// PythonConfig returns the embedded python.Config struct from the interface
func (c *Config) PythonConfig() *python.Config {
	return c.pyConf
}

// Validate will check the config for correctness.
func (c *Config) Validate() error {
	if c.CollectApplicationMetrics != nil && *c.CollectApplicationMetrics && !c.IsMaster {
		return errors.New("Cannot collect application metrics from non-master endpoint")
	}
	switch c.ClusterType {
	case sparkYarn, sparkMesos, sparkStandalone:
		return nil
	default:
		return fmt.Errorf("required configuration clusterType '%s' is invalid", c.ClusterType)
	}
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.PyMonitor
}

// Configure configures and runs the plugin in python
func (m *Monitor) Configure(conf *Config) error {
	conf.pyConf = &python.Config{
		MonitorConfig: conf.MonitorConfig,
		Host:          conf.Host,
		Port:          conf.Port,
		ModuleName:    "spark_plugin",
		ModulePaths:   []string{collectd.MakePythonPluginPath("spark")},
		TypesDBPaths:  []string{collectd.DefaultTypesDBPath()},
		PluginConfig: map[string]interface{}{
			"Host":            conf.Host,
			"Port":            conf.Port,
			"Cluster":         string(conf.ClusterType),
			"Applications":    conf.CollectApplicationMetrics,
			"EnhancedMetrics": conf.EnhancedMetrics,
		},
	}

	if conf.IsMaster {
		conf.pyConf.PluginConfig["Master"] = fmt.Sprintf("http://%s:%d", conf.Host, conf.Port)
		conf.pyConf.PluginConfig["MasterPort"] = conf.Port
	} else {
		conf.pyConf.PluginConfig["WorkerPorts"] = conf.Port
	}

	if conf.ClusterType != sparkYarn {
		conf.pyConf.PluginConfig["MetricsURL"] = fmt.Sprintf("http://%s", conf.Host)
	}

	return m.PyMonitor.Configure(conf)
}
