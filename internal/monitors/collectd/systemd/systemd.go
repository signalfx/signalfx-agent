// +build linux

package systemd

import (
	"strings"

	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
)

const (
	activeState = "ActiveState"
	subState    = "SubState"
	loadState   = "LoadState"
)

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
	// Systemd services to report on
	Services []string `yaml:"services" validate:"required"`
	// Flag for sending metrics about the state of systemd services
	SendActiveState bool `yaml:"sendActiveState"`
	// Flag for sending more detailed metrics about the state of systemd services
	SendSubState bool `yaml:"sendSubState"`
	// Flag for sending metrics about the load state of systemd services
	SendLoadState bool `yaml:"sendLoadState"`
}

// PythonConfig returns the embedded python.Config struct from the interface
func (c *Config) PythonConfig() *python.Config {
	return c.pyConf
}

// GetExtraMetrics returns additional metrics to allow through.
func (c *Config) GetExtraMetrics() []string {
	extraMetrics := make([]string, 0)
	for _, serviceState := range [...]string{activeState, subState, loadState} {
		for _, metric := range groupMetricsMap[serviceState] {
			if !includedMetrics[metric] && ((serviceState == activeState && c.SendActiveState) || (serviceState == subState && c.SendSubState) || (serviceState == loadState && c.SendLoadState)) {
				extraMetrics = append(extraMetrics, metric)
			}
		}
	}
	return extraMetrics
}

func (c *Config) services() (services []string) {
	for _, service := range c.Services {
		services = append(services, strings.Trim(service, " "))
	}
	return
}

func (c *Config) serviceStates() (serviceStates []string) {
	serviceStates = append(serviceStates, subState)
	if c.SendActiveState {
		serviceStates = append(serviceStates, activeState)
	}
	if c.SendLoadState {
		serviceStates = append(serviceStates, loadState)
	}
	return
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.PyMonitor
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	conf.pyConf = &python.Config{
		MonitorConfig: conf.MonitorConfig,
		ModuleName:    "collectd_systemd",
		ModulePaths:   []string{collectd.MakePythonPluginPath("systemd")},
		TypesDBPaths:  []string{collectd.DefaultTypesDBPath()},
		PluginConfig: map[string]interface{}{
			"Service":       conf.services(),
			"Interval":      conf.IntervalSeconds,
			"Verbose":       false,
			"ServiceStates": conf.serviceStates(),
		},
	}
	return m.PyMonitor.Configure(conf)
}
