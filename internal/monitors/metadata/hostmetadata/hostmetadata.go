package hostmetadata

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/metadata/aws/ec2metadata"
	"github.com/signalfx/golib/metadata/hostmetadata"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/metadata"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

const (
	monitorType      = "host-metadata"
	uptimeMetricName = "gauge.sf.host-plugin_uptime"
)

// MONITOR(host-metadata): This monitor collects metadata about a host.  It is
// required for some views in SignalFx to operate.

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{Monitor: &metadata.Monitor{}} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"false"`
	// The path to the proc filesystem. Useful to override in containerized
	// environments.
	ProcFSPath string `yaml:"procFSPath" default:"/proc"`
	// The path to the main host config dir. Userful to override in
	// containerized environments.
	EtcPath string `yaml:"etcPath" default:"/etc"`
}

// Monitor for host-metadata
type Monitor struct {
	*metadata.Monitor
	startTime time.Time
	cancel    func()
}

// Configure is the main function of the monitor, it will report host metadata
// on a varied interval
func (m *Monitor) Configure(conf *Config) error {
	intervals := []time.Duration{
		// 0-60 seconds
		time.Duration(rand.Int63n(60)) * time.Second,
		// 1 minute after the previous because sometimes pieces of metadata aren't available immediately on startup
		time.Duration(60) * time.Second,
		// 1 hour after the previous with some dither
		time.Duration(rand.Int63n(60)+3600) * time.Second,
		// 1 day after the previous with some dither
		time.Duration(rand.Int63n(600)+86400) * time.Second,
	}

	// set plugin start time
	m.startTime = time.Now()

	// set HOST_PROC and HOST_ETC for gopsutil
	if conf.ProcFSPath != "" {
		if err := os.Setenv("HOST_PROC", conf.ProcFSPath); err != nil {
			logger.Errorf("Error setting HOST_PROC env var %v", err)
		}
	}
	if conf.EtcPath != "" {
		if err := os.Setenv("HOST_ETC", conf.EtcPath); err != nil {
			logger.Errorf("Error setting HOST_ETC env var %v", err)
		}
	}

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	// gather metadata on intervals
	utils.RunOnIntervals(ctx,
		m.ReportMetadataProperties,
		intervals, utils.RepeatLast)

	// emit metadata metric
	utils.RunOnInterval(ctx,
		m.ReportUptimeMetric,
		time.Duration(conf.IntervalSeconds)*time.Second,
	)

	return nil
}

// info is an interface to the structs returned by the metadata packages in golib
type info interface {
	ToStringMap() map[string]string
}

// metadatafuncs are the functions to collect host metadata.
// putting them directly in the array raised issues with the return type of info
// By placing them inside of anonymous functions I can return (info, error)
var metadatafuncs = []func() (info, error){
	func() (info, error) { return hostmetadata.GetCPU() },
	func() (info, error) { return hostmetadata.GetMemory() },
	func() (info, error) { return hostmetadata.GetOS() },
	func() (info, error) { return ec2metadata.Get() },
}

var errNotAWS = fmt.Errorf("not an aws box")

// ReportMetadataProperties emits properties about the host
func (m *Monitor) ReportMetadataProperties() {
	for _, f := range metadatafuncs {
		meta, err := f()
		if err != nil && err != errNotAWS {
			logger.Error(err)
			continue
		}
		properties := meta.ToStringMap()
		for k, v := range properties {
			m.EmitProperty(k, v)
		}
	}
}

func (m *Monitor) getUptime(curr time.Time) int64 {
	return int64((curr.UnixNano() - m.startTime.UnixNano()) / 1000000)
}

// ReportUptimeMetric report metrics
func (m *Monitor) ReportUptimeMetric() {
	dims := map[string]string{
		"signalfx_agent": os.Getenv("SIGNALFX_AGENT_VERSION"),
	}

	if osInfo, err := hostmetadata.GetOS(); err == nil {
		if osInfo.HostLinuxVersion != "" {
			dims["linux"] = osInfo.HostLinuxVersion
		}
		if osInfo.HostKernelRelease != "" {
			dims["release"] = osInfo.HostKernelRelease
		}
		if osInfo.HostKernelVersion != "" {
			dims["version"] = osInfo.HostKernelVersion
		}
	}

	curr := time.Now()
	m.Output.SendDatapoint(
		datapoint.New(
			uptimeMetricName,
			dims,
			datapoint.NewIntValue(m.getUptime(curr)),
			datapoint.Gauge,
			curr,
		),
	)
}
