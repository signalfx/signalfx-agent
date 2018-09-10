// +build !windows

package jenkins

import (
	"os"
	"path/filepath"

	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
)

const monitorType = "collectd/jenkins"

// MONITOR(collectd/jenkins): Monitors jenkins by using the
// [jenkins collectd Python
// plugin](https://github.com/signalfx/collectd-jenkins), which collects
// metrics from jenkins instances
//
// Sample YAML configuration:
//
// ```yaml
// monitors:
// - type: collectd/jenkins
//   host: 127.0.0.1
//   port: 8080
//   metricsKey: reallylongmetricskey
// ```
//
// Sample YAML configuration with specific enhanced metrics included
//
// ```yaml
// monitors:
// - type: collectd/jenkins
//   host: 127.0.0.1
//   port: 8080
//   metricsKey: reallylongmetricskey
//   includeMetrics:
//   - "vm.daemon.count"
//   - "vm.terminated.count"
// ```
//
// Sample YAML configuration with all enhanced metrics included
//
// ```yaml
// monitors:
// - type: collectd/jenkins
//   host: 127.0.0.1
//   port: 8080
//   metricsKey: reallylongmetricskey
//   enhancedMetrics: true
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
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	// By not embedding python.Config we can override struct fields (i.e. Host and Port)
	// and add monitor specific config doc and struct tags.
	pyConfig *python.Config
	Host     string `yaml:"host" validate:"required"`
	Port     uint16 `yaml:"port" validate:"required"`
	// Key required for collecting metrics.  The access key located at
	// `Manage Jenkins > Configure System > Metrics > ADD.`
	// If empty, click `Generate`.
	MetricsKey string `yaml:"metricsKey" validate:"required"`
	// Whether to enable enhanced metrics
	EnhancedMetrics bool `yaml:"enhancedMetrics"`
	// Used to enable individual enhanced metrics when `enhancedMetrics` is
	// false
	IncludeMetrics []string `yaml:"includeMetrics"`
	// User with security access to jenkins
	Username string `yaml:"username"`
	// API Token of the user
	APIToken string `yaml:"apiToken" neverLog:"true"`
	// Path to the keyfile
	SSLKeyFile string `yaml:"sslKeyFile"`
	// Path to the certificate
	SSLCertificate string `yaml:"sslCertificate"`
	// Path to the ca file
	SSLCACerts string `yaml:"sslCACerts"`
}

// PythonConfig returns the python.Config struct contained in the config struct
func (c *Config) PythonConfig() *python.Config {
	return c.pyConfig
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.Monitor
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	pConf := map[string]interface{}{
		"Host":            conf.Host,
		"Port":            conf.Port,
		"Interval":        conf.IntervalSeconds,
		"MetricsKey":      conf.MetricsKey,
		"EnhancedMetrics": conf.EnhancedMetrics,
	}
	if conf.Username != "" {
		pConf["Username"] = conf.Username
	}
	if conf.APIToken != "" {
		pConf["APIToken"] = conf.APIToken
	}
	if conf.SSLKeyFile != "" {
		pConf["ssl_keyfile"] = conf.SSLKeyFile
	}
	if conf.SSLCertificate != "" {
		pConf["ssl_certificate"] = conf.SSLCertificate
	}
	if conf.SSLCACerts != "" {
		pConf["ssl_ca_certs"] = conf.SSLCACerts
	}
	if len(conf.IncludeMetrics) > 0 {
		pConf["IncludeMetric"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.IncludeMetrics,
		}
	}

	conf.pyConfig = &python.Config{
		MonitorConfig: conf.MonitorConfig,
		Host:          conf.Host,
		Port:          conf.Port,
		ModuleName:    "jenkins",
		ModulePaths:   []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "jenkins")},
		TypesDBPaths:  []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "types.db")},
		PluginConfig:  pConf,
	}
	return m.Monitor.Configure(conf)
}
