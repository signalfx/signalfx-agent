// +build !windows

package redis

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/core/config"

	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
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
			python.Monitor{
				MonitorCore: pyrunner.New("sfxcollectd"),
			},
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
	pyConf               *python.Config
	Host                 string `yaml:"host" validate:"required"`
	Port                 uint16 `yaml:"port" validate:"required"`
	// The name for the node is a canonical identifier which is used as plugin
	// instance. It is limited to 64 characters in length.  (**default**: "{host}:{port}")
	Name string `yaml:"name"`
	// Password to use for authentication.
	Auth string `yaml:"auth" neverLog:"true"`
	// Specify a pattern of keys to lists for which to send their length as a
	// metric. See below for more details.
	SendListLengths []ListLength `yaml:"sendListLengths"`
}

// PythonConfig returns the embedded python.Config struct from the interface
func (c *Config) PythonConfig() *python.Config {
	return c.pyConf
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.Monitor
}

// Configure configures and runs the plugin in collectd
func (rm *Monitor) Configure(conf *Config) error {

	instanceID := conf.Name
	if instanceID == "" {
		instanceID = fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	}

	conf.pyConf = &python.Config{
		MonitorConfig: conf.MonitorConfig,
		Host:          conf.Host,
		Port:          conf.Port,
		ModuleName:    "redis_info",
		ModulePaths:   []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "redis")},
		TypesDBPaths:  []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "types.db")},
		PluginConfig: map[string]interface{}{
			"Host":                                 conf.Host,
			"Port":                                 conf.Port,
			"Instance":                             instanceID,
			"Verbose":                              false,
			"Redis_uptime_in_seconds":              "gauge",
			"Redis_used_cpu_sys":                   "counter",
			"Redis_used_cpu_user":                  "counter",
			"Redis_used_cpu_sys_children":          "counter",
			"Redis_used_cpu_user_children":         "counter",
			"Redis_uptime_in_days":                 "gauge",
			"Redis_lru_clock":                      "counter",
			"Redis_connected_clients":              "gauge",
			"Redis_client_longest_output_list":     "gauge",
			"Redis_client_biggest_input_buf":       "gauge",
			"Redis_blocked_clients":                "gauge",
			"Redis_expired_keys":                   "counter",
			"Redis_evicted_keys":                   "counter",
			"Redis_rejected_connections":           "counter",
			"Redis_used_memory":                    "bytes",
			"Redis_used_memory_rss":                "bytes",
			"Redis_used_memory_peak":               "bytes",
			"Redis_used_memory_lua":                "bytes",
			"Redis_mem_fragmentation_ratio":        "gauge",
			"Redis_changes_since_last_save":        "gauge",
			"Redis_instantaneous_ops_per_sec":      "gauge",
			"Redis_rdb_bgsave_in_progress":         "gauge",
			"Redis_total_connections_received":     "counter",
			"Redis_total_commands_processed":       "counter",
			"Redis_total_net_input_bytes":          "counter",
			"Redis_total_net_output_bytes":         "counter",
			"Redis_keyspace_hits":                  "derive",
			"Redis_keyspace_misses":                "derive",
			"Redis_latest_fork_usec":               "gauge",
			"Redis_connected_slaves":               "gauge",
			"Redis_repl_backlog_first_byte_offset": "gauge",
			"Redis_master_repl_offset":             "gauge",
			"Redis_db0_keys":                       "gauge",
			"Redis_db0_expires":                    "gauge",
			"Redis_db0_avg_ttl":                    "gauge",
		},
	}
	if conf.Auth != "" {
		conf.pyConf.PluginConfig["Auth"] = conf.Auth
	}
	if conf.Name != "" {
		conf.pyConf.PluginConfig["Name"] = conf.Name
	}
	if len(conf.SendListLengths) > 0 {
		conf.pyConf.PluginConfig["SendListLength"] = conf.SendListLengths
	}

	return rm.Monitor.Configure(conf)
}
