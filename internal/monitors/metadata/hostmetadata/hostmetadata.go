package hostmetadata

import (
	"context"
	"math/rand"
	"os"
	"time"

	"github.com/signalfx/signalfx-agent/internal/core/common/constants"

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
	errNotAWS        = "not an aws box"
)

// MONITOR(host-metadata): This monitor collects metadata properties about a
// host.  It is required for some views in SignalFx to operate.
//
// ```yaml
// monitors:
//   - type: host-metadata
// ```
//
// In containerized environments host `/etc` and `/proc` may not be located
// directly under the root path.  You can specify the path to `proc` and `etc`
// using the monitor configurations `procFSPath` and `etcPath`
//
// ```yaml
// monitors:
//   - type: host-metadata
//     procFSPath: "/hostfs/proc"
//     etcPath: "/hostfs/etc"
// ```
//
// GAUGE(gauge.sf.host-plugin_uptime): The time this monitor has been running in
// seconds.  Dimensions include `signalfx_agent`, `collectd`, `kernel_release`,
// `kernel_version`, and `kernel_name`

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
	// The path to the main host config dir. Useful to override in
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
		// on startup with some 0-60s dither
		time.Duration(rand.Int63n(60)) * time.Second,
		// 1 minute after the previous because sometimes pieces of metadata
		// aren't available immediately on startup like aws identity information
		time.Duration(60) * time.Second,
		// 1 hour after the previous with some 0-60s dither
		time.Duration(rand.Int63n(60)+3600) * time.Second,
		// 1 day after the previous with some 0-10m dither
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

	logger.Debugf("Waiting %f seconds to emit metadata", intervals[0].Seconds())

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

// ReportMetadataProperties emits properties about the host
func (m *Monitor) ReportMetadataProperties() {
	for _, f := range metadatafuncs {
		meta, err := f()

		if err != nil {
			// suppress the not an aws box error message it is expected
			if err.Error() == errNotAWS {
				logger.Debug(err)
			} else {
				logger.Error(err)
			}
			continue
		}

		// get the properties as a map
		properties := meta.ToStringMap()

		// emit each key/value pair
		for k, v := range properties {
			m.EmitProperty(k, v)
		}
	}
}

// ReportUptimeMetric report metrics
func (m *Monitor) ReportUptimeMetric() {
	dims := map[string]string{
		"plugin":         monitorType,
		"signalfx_agent": os.Getenv(constants.AgentVersionEnvVar),
	}

	if collectdVersion := os.Getenv(constants.CollectdVersionEnvVar); collectdVersion != "" {
		dims["collectd"] = collectdVersion
	}

	if osInfo, err := hostmetadata.GetOS(); err == nil {
		switch osInfo.HostKernelName {
		case "windows":
			dims["windows"] = osInfo.HostKernelRelease
		case "linux":
			dims["linux"] = osInfo.HostLinuxVersion
		}
		dims["kernel_name"] = osInfo.HostKernelName
		dims["kernel_release"] = osInfo.HostKernelRelease
		dims["kernel_version"] = osInfo.HostKernelVersion
	}

	m.Output.SendDatapoint(
		datapoint.New(
			uptimeMetricName,
			dims,
			datapoint.NewFloatValue(time.Since(m.startTime).Seconds()),
			datapoint.Gauge,
			time.Now(),
		),
	)
}
