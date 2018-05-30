// +build !windows

package jenkins

//go:generate collectd-template-to-go jenkins.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
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
			*collectd.NewMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	Host                 string `yaml:"host" validate:"required"`
	Port                 uint16 `yaml:"port" validate:"required"`
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
	APIToken string `yaml:"apiToken"`
	// Path to the keyfile
	SSLKeyFile string `yaml:"sslKeyFile"`
	// Path to the certificate
	SSLCertificate string `yaml:"sslCertificate"`
	// Path to the ca file
	SSLCACerts string `yaml:"sslCACerts"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Validate will check the config for correctness.
func (c *Config) Validate() error {

	return nil
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) error {
	return am.SetConfigurationAndRun(conf)
}
