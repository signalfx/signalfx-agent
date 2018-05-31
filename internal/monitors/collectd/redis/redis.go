// +build !windows

package redis

//go:generate collectd-template-to-go redis.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/redis"

// MONITOR(collectd/redis): Monitors a redis instance using the [collectd
// Python Redis plugin](https://github.com/signalfx/redis-collectd-plugin).
//
// See the [integrations
// doc](https://github.com/signalfx/integrations/tree/master/collectd-redis)
// for more information.
//
// Sample YAML configuration:
//
// ```yaml
// monitors:
// - type: collectd/redis
//   host: 127.0.0.1
//   port: 9100
// ```
//
// Sample YAML configuration with list lengths:
//
// ```yaml
// monitors:
// - type: collectd/redis
//   host: 127.0.0.1
//   port: 9100
//   sendListLengths:
//   - databaseIndex: 0
//     keyPattern: 'mylist*'
// ```
//

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			*collectd.NewMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// ListLength defines a database index and key pattern for sending list lengths
type ListLength struct {
	// The database index.
	DBIndex uint16 `yaml:"databaseIndex" validate:"required"`
	// Can be a globbed pattern (only * is supported), in which case all keys
	// matching that glob will be processed.  The pattern should be placed in
	// single quotes (').  Ex. `'mylist*'`
	KeyPattern string `yaml:"keyPattern" validate:"required"`
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`

	Host string `yaml:"host" validate:"required"`
	Port uint16 `yaml:"port" validate:"required"`
	// The name for the node is a canonical identifier which is used as plugin
	// instance. It is limited to 64 characters in length.  (**default**: "{host}:{port}")
	Name string `yaml:"name"`
	// Password to use for authentication.
	Auth string `yaml:"auth" neverLog:"true"`
	// Specify a pattern of keys to lists for which to send their length as a
	// metric. See below for more details.
	SendListLengths []ListLength `yaml:"sendListLengths"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (rm *Monitor) Configure(conf *Config) error {
	return rm.SetConfigurationAndRun(conf)
}
