package couchbase

import (
	"errors"

	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"

	"github.com/signalfx/signalfx-agent/internal/core/config"

	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"

	"github.com/signalfx/signalfx-agent/internal/monitors"
)

const monitorType = "collectd/couchbase"

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
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	pyConf               *python.Config
	Host                 string `yaml:"host" validate:"required"`
	Port                 uint16 `yaml:"port" validate:"required"`
	// Define what this Module block will monitor: "NODE", for a Couchbase node,
	// or "BUCKET" for a Couchbase bucket.
	CollectTarget string `yaml:"collectTarget" validate:"required"`
	// If CollectTarget is "BUCKET", CollectBucket specifies the name of the
	// bucket that this will monitor.
	CollectBucket string `yaml:"collectBucket"`
	// Name of this Couchbase cluster. (**default**:"default")
	ClusterName string `yaml:"clusterName"`
	// Change to "detailed" to collect all available metrics from Couchbase
	// stats API. Defaults to "default", collecting a curated set that works
	// well with SignalFx. See [metric_info.py](https://github.com/signalfx/collectd-couchbase/blob/master/metric_info.py) for more information.
	CollectMode string `yaml:"collectMode"`
	// Username to authenticate with
	Username string `yaml:"username"`
	// Password to authenticate with
	Password string `yaml:"password" neverLog:"true"`
}

// PythonConfig returns the embedded python.Config struct from the interface
func (c *Config) PythonConfig() *python.Config {
	return c.pyConf
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.PyMonitor
}

// Validate will check the config for correctness.
func (c *Config) Validate() error {
	if c.CollectTarget == "BUCKET" && c.CollectBucket == "" {
		return errors.New(
			"CollectBucket must be configured when CollectTarget is set to BUCKET")
	}
	return nil
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	conf.pyConf = &python.Config{
		MonitorConfig: conf.MonitorConfig,
		Host:          conf.Host,
		Port:          conf.Port,
		ModuleName:    "couchbase",
		ModulePaths:   []string{collectd.MakePath("couchbase")},
		TypesDBPaths:  []string{collectd.MakePath("types.db")},
		PluginConfig: map[string]interface{}{
			"Host":          conf.Host,
			"Port":          conf.Port,
			"CollectTarget": conf.CollectTarget,
			"Interval":      conf.IntervalSeconds,
			"FieldLength":   1024,
			"CollectBucket": conf.CollectBucket,
			"ClusterName":   conf.ClusterName,
			"CollectMode":   conf.CollectMode,
			"Username":      conf.Username,
			"Password":      conf.Password,
		},
	}

	return m.PyMonitor.Configure(conf)
}
