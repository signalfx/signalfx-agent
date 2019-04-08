package appmesh

import (
	"fmt"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/statsd"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
)

func init() {
	monitors.Register(monitorMetadata.MonitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"false" singleInstance:"false"`
	// The host/address on which to bind the UDP listener that accepts statsd
	// datagrams
	ListenAddress string `yaml:"listenAddress" default:"localhost"`
	// The port on which to listen for statsd messages
	ListenPort uint16 `yaml:"listenPort" default:"8125"`
	// A prefix in metric names that needs to be removed before metric name conversion
	MetricPrefix string `yaml:"metricPrefix"`
}

// Monitor that listens to incoming statsd metrics and converts the metrics in AWS AppMesh metric format
type Monitor struct {
	Output  types.Output
	monitor *statsd.Monitor
}

// Configure the monitor and kick off volume metric syncing
func (m *Monitor) Configure(conf *Config) error {
	var err error
	m.monitor, err = m.statsDMonitor(conf)

	if err != nil {
		return fmt.Errorf("could not start StatsD monitor: %v", err)
	}

	return nil
}

func (m *Monitor) statsDMonitor(conf *Config) (*statsd.Monitor, error) {
	monitor := &statsd.Monitor{Output: m.Output.Copy()}

	return monitor, monitor.Configure(&statsd.Config{
		MonitorConfig: conf.MonitorConfig,
		ListenAddress: conf.ListenAddress,
		ListenPort:    conf.ListenPort,
		MetricPrefix:  conf.MetricPrefix,
		Converters: []statsd.ConverterInput{
			{
				Pattern:    "cluster.cds_{traffic}_{mesh}_{service}-vn_{}.{action}",
				MetricName: "{action}",
			},
		},
	})
}

// Shutdown shuts down the internal StatsD monitor
func (m *Monitor) Shutdown() {
	if m.monitor != nil {
		m.monitor.Shutdown()
	}
}
