// +build !windows

package protocols

//go:generate collectd-template-to-go protocols.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/protocols"

// MONITOR(collectd/protocols): Gathers metrics about the network protocol
// stacks running on the system by using the [collectd protocols
// plugin](https://collectd.org/wiki/index.php/Plugin:Protocols).
//
// See the [integrations
// doc](https://github.com/signalfx/integrations/tree/master/collectd-protocols)
// for more information.

// CUMULATIVE(protocol_counter.ActiveOpens): The number of times TCP connections transitioned from the CLOSED state to the SYN-SENT state.

// CUMULATIVE(protocol_counter.CurrEstab): The number of TCP connections currently in either ESTABLISHED or CLOSE-WAIT state.

// CUMULATIVE(protocol_counter.DelayedACKs): The number of acknowledgements delayed by TCP Delayed Acknowledgement

// CUMULATIVE(protocol_counter.InDestUnreachs): The number of ICMP Destination Unreachable messages received

// CUMULATIVE(protocol_counter.PassiveOpens): The number of times that a server opened a connection, due to receiving a TCP SYN packet.

// CUMULATIVE(protocol_counter.RetransSegs): The total number of segments retransmitted

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
