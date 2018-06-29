package hostmetadata

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/metadata/hostmetadata"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/metadata"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

const monitorType = "host-metadata"

// MONITOR(host-metadata): This monitor collects metadata about a host.  It is
// required for some views in SignalFx to operate.

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{MetadataMonitor: &metadata.MetadataMonitor{}} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"false"`
}

// Configure is the main function of the monitor, it will report host metadata
// on a varried interval
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

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	// gather metadata on intervals
	utils.RunOnIntervals(ctx,
		m.ReportMetadataProperties,
		intervals, utils.RepeatLast)

	// emit metadata metric
	utils.RunOnInterval(ctx,
		ReportUptimeMetric,
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
	// func() (info, error) { i, err := hostmetadata.GetMemory(); return i, err },
	// func() (info, error) { i, err := hostmetadata.GetOS(); return i, err },
	// func() (info, error) { i, err := ec2metadata.Get(); return i, err },
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

const uptimeMetricName = "sf.host-plugin_uptime"

// ReportUptimeMetric report metrics
func (m *Monitor) ReportUptimeMetric() {
	dims = map[string]string{}
	if osInfo, err := hostmetadata.GetOS(); err == nil {
		if osInfo.HostLinuxVersion != "" {
			dims["linux"] = osInfo.HostLinuxVersion
		}
		// if osInfo.HostKernelRelease != "" {
		// osInfo.HostKernelRelease != "" {
		// 	dims["release"] = osInfo.HostKernelRelease
		// }
	}
	m.Output.SendDatapoint(
		datapoint.New(
			uptimeMetricName,
			dims,
			getUptime(),
			datapoint.Gauge,
			time.Now(),
		),
	)
}

func (m *Monitor) getUptime() {
	return (time.Now() - m.startTime).Nanosecond() / 1000000
}

// Monitor for Docker
type Monitor struct {
	*metadata.MetadataMonitor
	starTime time.Time
	cancel   func()
}
