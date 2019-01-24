package hostmetadata

import (
	"context"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/metadata/aws/ec2metadata"
	"github.com/signalfx/golib/metadata/hostmetadata"
	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/metadata"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

const (
	monitorType      = "host-metadata"
	errNotAWS        = "not an aws box"
	uptimeMetricName = "sfxagent.hostmetadata"
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
// using the top level agent configurations `procPath` and `etcPath`
//
// ```yaml
// procPath: /proc
// etcPath: /etc
// monitors:
//   - type: host-metadata
// ```
//
// Metadata updates occur on a sparse interval of approximately
// 1m, 1m, 1h, 1d and continues repeating once per day.
// Setting the `Interval` configuration for this monitor will not affect the
// sparse interval on which metadata is collected.
//
// GAUGE(sfxagent.hostmetadata): The time the hostmetadata monitor has been
// running in seconds.  It includes dimensional metadata about the host and
// agent.
//
// DIMENSION(signalfx_agent): The version of the signalfx-agent
// DIMENSION(collectd): The version of collectd in the signalfx-agent
// DIMENSION(kernel_name): The name of the host kernel.
// DIMENSION(kernel_version): The version of the host kernel.
// DIMENSION(kernel_release): The release of the host kernel.
// DIMENSION(os_version): The version of the os on the host.

// the time that the agent started / imported this package
var startTime = time.Now()

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{Monitor: metadata.Monitor{}} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `singleInstance:"true" acceptsEndpoints:"false"`
}

// Monitor for host-metadata
type Monitor struct {
	metadata.Monitor
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

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	logger.Debugf("Waiting %f seconds to emit metadata", intervals[0].Seconds())

	// gather metadata on intervals
	utils.RunOnArrayOfIntervals(ctx,
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
		kernelName := strings.ToLower(osInfo.HostKernelName)
		dims["kernel_name"] = kernelName
		dims["kernel_release"] = osInfo.HostKernelRelease
		dims["kernel_version"] = osInfo.HostKernelVersion

		switch kernelName {
		case "windows":
			dims["os_version"] = osInfo.HostKernelRelease
		case "linux":
			dims["os_version"] = osInfo.HostLinuxVersion
		}
	}

	m.Output.SendDatapoint(
		datapoint.New(
			uptimeMetricName,
			dims,
			datapoint.NewFloatValue(time.Since(startTime).Seconds()),
			datapoint.Counter,
			time.Now(),
		),
	)
}
