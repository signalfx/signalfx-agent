// +build linux

package systemd

import (
	"fmt"
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

type stringSet map[string]bool

// UnmarshalYAML is used to unmarshal into stringSet
func (s *stringSet) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*s = stringSet{}
	var (
		keys []string
		err  error
	)
	if err = unmarshal(&keys); err == nil && len(keys) > 0 {
		for _, key := range keys {
			if key = strings.Trim(key, " "); key != "" {
				(*s)[key] = true
			}
		}
	}
	return err
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	pyConf               *python.Config
	// Systemd services to report on
	Services []string `yaml:"services" validate:"required"`
	// Systemd service states. The default state is `ActiveState` and the default metric exported is `gauge.substate.running`. Possible service states are `ActiveState`, `SubState` and `LoadState`
	ServiceStates stringSet `yaml:"serviceStates"`
}

// PythonConfig returns the embedded python.Config struct from the interface
func (c *Config) PythonConfig() *python.Config {
	return c.pyConf
}

// Validate validates optional configured `serviceStates`. Possible values are `ActiveState`, `SubState` and `LoadState`
func (c *Config) Validate() error {
	for serviceState := range c.ServiceStates {
		switch serviceState {
		case activeState, subState, loadState:
		default:
			return fmt.Errorf("%s is an invalid service states. Possible service states are %s, %s, %s", serviceState, activeState, subState, loadState)
		}
	}
	return nil
}

// GetExtraMetrics returns additional metrics to allow through.
func (c *Config) GetExtraMetrics() []string {
	extraMetrics := make([]string, 0)
	for serviceState := range c.ServiceStates {
		for _, metric := range groupMetricsMap[serviceState] {
			if !includedMetrics[metric] {
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
	for serviceState := range c.ServiceStates {
		serviceStates = append(serviceStates, serviceState)
	}
	if c.ServiceStates[activeState] == false {
		serviceStates = append(serviceStates, activeState)
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
