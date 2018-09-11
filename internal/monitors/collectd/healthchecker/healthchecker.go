// +build !windows

package healthchecker

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/core/config"

	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"

	"github.com/signalfx/signalfx-agent/internal/monitors"
)

const monitorType = "collectd/health-checker"

// MONITOR(collectd/health-checker): A simple Collectd Python-based monitor
// that hits an endpoint and checks if the configured JSON value is returned in
// the response body.

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
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	pyConf               *python.Config
	Host                 string `yaml:"host" validate:"required"`
	Port                 uint16 `yaml:"port" validate:"required"`
	Name                 string `yaml:"name"`
	// The HTTP path that contains a JSON document to verify
	Path string `yaml:"path" default:"/"`
	// If `jsonKey` and `jsonVal` are given, the given endpoint will be
	// interpreted as a JSON document and will be expected to contain the given
	// key and value for the service to be considered healthy.
	JSONKey string `yaml:"jsonKey"`
	// This can be either a string or numeric type
	JSONVal interface{} `yaml:"jsonVal"`
	// If true, the endpoint will be connected to on HTTPS instead of plain
	// HTTP.  It is invalid to specify this if `tcpCheck` is true.
	UseHTTPS bool `yaml:"useHTTPS"`
	// If true, and `useHTTPS` is true, the server's SSL/TLS cert will not be
	// verified.
	SkipSecurity bool `yaml:"skipSecurity"`
	// If true, the plugin will verify that it can connect to the given
	// host/port value. JSON checking is not supported.
	TCPCheck bool `yaml:"tcpCheck"`
}

// PythonConfig returns the embedded python.Config struct from the interface
func (c *Config) PythonConfig() *python.Config {
	return c.pyConf
}

// Validate the given config
func (c *Config) Validate() error {
	if c.TCPCheck && (c.SkipSecurity || c.UseHTTPS) {
		return errors.New("neither skipSecurity nor useHTTPS should be set when tcpCheck is true")
	}
	if c.TCPCheck && (c.JSONKey != "" || c.JSONVal != nil) {
		return errors.New("cannot do JSON value check with tcpCheck set to true")
	}
	return nil
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.Monitor
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	conf.pyConf = &python.Config{
		MonitorConfig: conf.MonitorConfig,
		Host:          conf.Host,
		Port:          conf.Port,
		ModuleName:    "health_checker",
		ModulePaths:   []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "health_checker")},
		TypesDBPaths:  []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "types.db")},
		PluginConfig: map[string]interface{}{
			"Instance": conf.Name,
		},
	}

	if conf.TCPCheck {
		conf.pyConf.PluginConfig["URL"] = conf.Host
		conf.pyConf.PluginConfig["TCP"] = conf.Port
	} else {
		conf.pyConf.PluginConfig["URL"] = "http{{if .UseHTTPS}}s{{end}}://{{.Host}}:{{.Port}}:{{.Path}}"
		conf.pyConf.PluginConfig["SkipSecurity"] = conf.SkipSecurity
	}

	if conf.JSONKey != "" {
		conf.pyConf.PluginConfig["JSONKey"] = conf.JSONKey
	}

	if conf.JSONVal != nil {
		conf.pyConf.PluginConfig["JSONVal"] = conf.JSONVal
	}

	return m.Monitor.Configure(conf)
}
