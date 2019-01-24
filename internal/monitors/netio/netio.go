package netio

import (
	"context"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/net"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/filter"
	log "github.com/sirupsen/logrus"
)

const monitorType = "net-io"

// setting net.IOCounters to a package variable for testing purposes
var iOCounters = net.IOCounters

// MONITOR(net-io):
// This monitor reports I/O metrics about network interfaces.
//
// On Linux hosts, this monitor relies on the `/proc` filesystem.
// If the underlying host's `/proc` file system is mounted somewhere other than
// /proc please specify the path using the top level configuration `procPath`.
//
// ```yaml
// procPath: /proc
// monitors:
//  - type: net-io
// ```

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `singleInstance:"false" acceptsEndpoints:"false"`
	// The interfaces to include/exclude, is interpreted as a regex if
	// surrounded by `/`.
	Interfaces []string `yaml:"interfaces" default:"[\"*\", \"!/^lo\\\\d*$/\", \"!/^docker.*/\", \"!/^t(un|ap)\\\\d*$/\", \"!/^veth.*$/\", \"!/^Loopback*/\"]"`
}

// Monitor for Utilization
type Monitor struct {
	Output                 types.Output
	cancel                 func()
	conf                   *Config
	filter                 *filter.ExhaustiveStringFilter
	networkTotal           uint64
	previousInterfaceStats map[string]*net.IOCountersStat
}

func (m *Monitor) updateTotals(pluginInstance string, intf *net.IOCountersStat) {
	prev, ok := m.previousInterfaceStats[pluginInstance]

	// update total received
	// if there's a previous value and the counter didn't reset
	if ok && prev.BytesRecv >= 0 && intf.BytesRecv >= prev.BytesRecv {
		m.networkTotal += (intf.BytesRecv - prev.BytesRecv)
	} else {
		m.networkTotal += intf.BytesRecv
	}

	// update total sent
	// if there's a previous value and the counter didn't reset
	if ok && prev.BytesSent >= 0 && intf.BytesSent >= prev.BytesSent {
		m.networkTotal += (intf.BytesSent - prev.BytesSent)
	} else {
		m.networkTotal += intf.BytesSent
	}

	// store values for reference next interval
	m.previousInterfaceStats[pluginInstance] = intf
}

// EmitDatapoints emits a set of memory datapoints
func (m *Monitor) EmitDatapoints() {
	info, err := iOCounters(true)
	if err != nil {
		logger.Errorf(err.Error())
	}
	for _, intf := range info {
		// skip it if the interface doesn't match
		if !m.filter.Matches(intf.Name) {
			logger.Debugf("skipping interface '%s'", intf.Name)
			continue
		}

		pluginInstance := strings.Replace(intf.Name, " ", "_", -1)

		m.updateTotals(pluginInstance, &intf)

		dimensions := map[string]string{"plugin": monitorType, "plugin_instance": pluginInstance, "interface": pluginInstance}

		// if_errors.rx
		m.Output.SendDatapoint(datapoint.New("if_errors.rx", dimensions, datapoint.NewIntValue(int64(intf.Errin)), datapoint.Counter, time.Time{}))

		// if_errors.tx
		m.Output.SendDatapoint(datapoint.New("if_errors.tx", dimensions, datapoint.NewIntValue(int64(intf.Errout)), datapoint.Counter, time.Time{}))

		// if_octets.rx
		m.Output.SendDatapoint(datapoint.New("if_octets.rx", dimensions, datapoint.NewIntValue(int64(intf.BytesRecv)), datapoint.Counter, time.Time{}))

		// if_octets.tx
		m.Output.SendDatapoint(datapoint.New("if_octets.tx", dimensions, datapoint.NewIntValue(int64(intf.BytesSent)), datapoint.Counter, time.Time{}))

		// if_packets.rx
		m.Output.SendDatapoint(datapoint.New("if_packets.rx", dimensions, datapoint.NewIntValue(int64(intf.PacketsRecv)), datapoint.Counter, time.Time{}))

		// if_packets.tx
		m.Output.SendDatapoint(datapoint.New("if_packets.tx", dimensions, datapoint.NewIntValue(int64(intf.PacketsSent)), datapoint.Counter, time.Time{}))
	}

	// network.total
	m.Output.SendDatapoint(datapoint.New("network.total", map[string]string{"plugin": types.UtilizationMetricPluginName}, datapoint.NewIntValue(int64(m.networkTotal)), datapoint.Counter, time.Time{}))
}

// Configure is the main function of the monitor, it will report host metadata
// on a varied interval
func (m *Monitor) Configure(conf *Config) error {
	if runtime.GOOS != "windows" {
		logger.Warningf("'%s' monitor is in beta on this platform.  For production environments please use 'collectd/%s'.", monitorType, monitorType)
	}
	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	m.conf = conf

	// initialize previous stats map and network total
	m.previousInterfaceStats = map[string]*net.IOCountersStat{}
	m.networkTotal = 0

	// configure filters
	var err error
	if len(conf.Interfaces) == 0 {
		m.filter, err = filter.NewExhaustiveStringFilter([]string{"*"})
		logger.Debugf("empty interface list, defaulting to '*'")
	} else {
		m.filter, err = filter.NewExhaustiveStringFilter(conf.Interfaces)
	}

	// return an error if we can't set the filter
	if err != nil {
		return err
	}

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		m.EmitDatapoints()
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
