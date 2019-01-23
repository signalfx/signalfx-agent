package netinterface

import (
	"context"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/net"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/gopsutilhelper"
	log "github.com/sirupsen/logrus"
)

const monitorType = "interface"

// setting net.IOCounters to a package variable for testing purposes
var iOCounters = net.IOCounters

// MONITOR(interface):
// This monitor reports network interface and network interface total metrics.
//
//
// ```yaml
// monitors:
//  - type: interface
// ```

// TODO: make ProcFSPath a global config

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

// Config for this monitor
type Config struct {
	config.MonitorConfig `singleInstance:"true" acceptsEndpoints:"false"`
	// The path to the proc filesystem. Useful to override in containerized
	// environments.  (Does not apply to windows)
	ProcFSPath string `yaml:"procFSPath" default:"/proc"`
	// If true, the interfaces selected by `selectInterfaces` will be
	// excluded and all others included.
	IgnoreSelected *bool `yaml:"ignoreSelected"`
	// The interfaces to include/exclude, is interpreted as a regex if
	// surrounded by `/`.
	Interfaces             []string `yaml:"interfaces" default:"[\"/^lo\\\\d*$/\", \"/^docker.*/\", \"/^t(un|ap)\\\\d*$/\", \"/^veth.*$/\", \"/^Loopback*/\"]"`
	interfaces             []*regexp.Regexp
	stringInterfaces       map[string]struct{}
	networkTotal           uint64
	previousInterfaceStats map[string]*net.IOCountersStat
}

func (m *Monitor) updateTotals(conf *Config, pluginInstance string, intf *net.IOCountersStat) {
	prev, ok := conf.previousInterfaceStats[pluginInstance]

	// update total received
	// if there's a previous value and the counter didn't reset
	if ok && prev.BytesRecv >= 0 && intf.BytesRecv >= prev.BytesRecv {
		conf.networkTotal += (intf.BytesRecv - prev.BytesRecv)
	} else {
		conf.networkTotal += intf.BytesRecv
	}

	// update total sent
	// if there's a previous value and the counter didn't reset
	if ok && prev.BytesSent >= 0 && intf.BytesSent >= prev.BytesSent {
		conf.networkTotal += (intf.BytesSent - prev.BytesSent)
	} else {
		conf.networkTotal += intf.BytesSent
	}

	// store values for reference next interval
	conf.previousInterfaceStats[pluginInstance] = intf
}

func shouldSkipInterface(conf *Config, intf *net.IOCountersStat) (shouldSkip bool) {
	// check for plain string match
	_, match := conf.stringInterfaces[intf.Name]
	// check for regex match
	match = match || utils.FindMatchString(intf.Name, conf.interfaces)
	if (match && *conf.IgnoreSelected) || (!match && !*conf.IgnoreSelected) {
		shouldSkip = true
	}
	return
}

// EmitDatapoints emits a set of memory datapoints
func (m *Monitor) EmitDatapoints(conf *Config) {
	info, err := iOCounters(true)
	if err != nil {
		logger.Errorf(err.Error())
	}
	for _, intf := range info {
		// handle selecting mountpoints
		if shouldSkipInterface(conf, &intf) {
			logger.Debugf("skipping interface '%s'", intf.Name)
			continue
		}

		pluginInstance := strings.Replace(intf.Name, " ", "_", -1)

		m.updateTotals(conf, pluginInstance, &intf)

		dimensions := map[string]string{"plugin": monitorType, "plugin_instance": pluginInstance}

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
	m.Output.SendDatapoint(datapoint.New("network.total", map[string]string{"plugin": constants.UtilizationMetricPluginName}, datapoint.NewIntValue(int64(conf.networkTotal)), datapoint.Counter, time.Time{}))
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

	// initialize previous stats map
	conf.previousInterfaceStats = map[string]*net.IOCountersStat{}

	// default to true when not set
	if conf.IgnoreSelected == nil {
		t := true
		conf.IgnoreSelected = &t
	}

	var errs []error
	conf.interfaces, conf.stringInterfaces, errs = utils.RegexpStringsToRegexp(conf.Interfaces)
	for _, err := range errs {
		logger.Errorf(err.Error())
	}

	// set HOST_PROC for gopsutil
	if err := gopsutilhelper.SetEnvVars(map[string]string{gopsutilhelper.HostProc: conf.ProcFSPath}); err != nil {
		return err
	}

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		m.EmitDatapoints(conf)
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel func()
}
