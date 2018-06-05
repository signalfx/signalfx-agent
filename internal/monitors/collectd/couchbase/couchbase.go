// +build !windows

package couchbase

//go:generate collectd-template-to-go couchbase.tmpl

import (
	"errors"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/couchbase"

// MONITOR(collectd/couchbase): Monitors couchbase by using the
// [couchbase collectd Python
// plugin](https://github.com/signalfx/collectd-couchbase), which collects
// metrics from couchbase instances
//
// Sample YAML configuration with custom query:
//
// ```yaml
// monitors:
// - type: collectd/couchbase
//   host: 127.0.0.1
//   port: 8091
//   collectTarget: "NODE"
//   clusterName: "my-cluster"
//   username: "user"
//   password: "password"
// ```

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			*collectd.NewMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
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

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
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
func (am *Monitor) Configure(conf *Config) error {
	return am.SetConfigurationAndRun(conf)
}
