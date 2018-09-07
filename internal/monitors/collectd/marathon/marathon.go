// +build !windows

package marathon

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/signalfx/signalfx-agent/internal/core/common/constants"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
)

const monitorType = "collectd/marathon"

// MONITOR(collectd/marathon): Monitors a Mesos Marathon instance using the
// [collectd Marathon Python plugin](https://github.com/signalfx/collectd-marathon).
//
// See the [integrations
// doc](https://github.com/signalfx/integrations/tree/master/collectd-marathon)
// for more information on configuration.
//
// Sample YAML configuration:
//
// ```yaml
// monitors:
//   - type: collectd/marathon
//     host: 127.0.0.1
//     port: 8080
//     scheme: http
// ```
//
// Sample YAML configuration for DC/OS:
//
// ```yaml
// monitors:
//   - type: collectd/marathon
//     host: 127.0.0.1
//     port: 8080
//     scheme: https
//     dcosAuthURL: https://leader.mesos/acs/api/v1/auth/login
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

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	// Make this single instance since we can't add dimensions
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true" singleInstance:"true"`
	// By not embedding python.Config we can override struct fields (i.e. Host and Port)
	// and add monitor specific config doc and struct tags.
	pyConfig *python.Config
	Host     string `yaml:"host" validate:"required"`
	Port     uint16 `yaml:"port" validate:"required"`
	// Username used to authenticate with Marathon.
	Username string `yaml:"username"`
	// Password used to authenticate with Marathon.
	Password string `yaml:"password" neverLog:"true"`
	// Set to either `http` or `https`.
	Scheme string `yaml:"scheme" default:"http"`
	// The dcos authentication URL which the plugin uses to get authentication
	// tokens from. Set scheme to "https" if operating DC/OS in strict mode and
	// dcosAuthURL to "https://leader.mesos/acs/api/v1/auth/login"
	// (which is the default DNS entry provided by DC/OS)
	DCOSAuthURL string `yaml:"dcosAuthURL"`
}

// PythonConfig returns the python.Config struct contained in the config struct
func (c *Config) PythonConfig() *python.Config {
	return c.pyConfig
}

// Validate config issues
func (c *Config) Validate() error {
	if c.DCOSAuthURL != "" && c.Scheme != "https" {
		return errors.New("Scheme must be set to https when using a DCOSAuthURL")
	}
	return nil
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.Monitor
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	// marathon's configuration is different, all configurations are
	// packed into an array of values for a given host
	host := []interface{}{conf.Scheme, conf.Host, conf.Port}
	if conf.Username != "" {
		host = append(host, conf.Username)
	}
	if conf.Password != "" {
		host = append(host, conf.Password)
	}
	if conf.DCOSAuthURL != "" {
		host = append(host, conf.DCOSAuthURL)
	}
	conf.pyConfig = &python.Config{
		MonitorConfig: conf.MonitorConfig,
		Host:          conf.Host,
		Port:          conf.Port,
		ModuleName:    "marathon",
		ModulePaths:   []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "marathon")},
		TypesDBPaths:  []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "types.db")},
		PluginConfig: map[string]interface{}{
			"host":    host,
			"verbose": false,
		},
	}
	return m.Monitor.Configure(conf)
}
