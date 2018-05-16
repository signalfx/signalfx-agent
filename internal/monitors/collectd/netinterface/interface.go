// +build !windows

// Package netinterface wraps the "interface" collectd plugin for gather
// network interface metrics.  It is called netinterface because "interface" is
// a keyword in golang.
package netinterface

//go:generate collectd-template-to-go interface.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/interface"

// MONITOR(collectd/interface): Collectd stats about network interfaces on the
// system by using the [collectd interface
// plugin](https://collectd.org/wiki/index.php/Plugin:Interface).
//
// See the [integrations
// doc](https://github.com/signalfx/integrations/tree/master/collectd-interface)
// for more information.

// CUMULATIVE(if_errors.rx): Count of receive errors on the interface
// CUMULATIVE(if_errors.tx): Count of transmit errors on the interface
// CUMULATIVE(if_octets.rx): Count of bytes (octets) received on the interface
// CUMULATIVE(if_octets.tx): Count of bytes (octets) transmitted by the interface
// CUMULATIVE(if_packets.rx): Count of packets received on the interface
// CUMULATIVE(if_packets.tx): Count of packets transmitted by the interface

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			MonitorCore: *collectd.NewMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `singleInstance:"true"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	return m.SetConfigurationAndRun(conf)
}
