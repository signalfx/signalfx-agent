package mongodbatlas

import (
	"context"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	"github.com/signalfx/signalfx-agent/pkg/utils/timeutil"
	"github.com/sirupsen/logrus"
	"sync/atomic"
	"time"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

var logger = logrus.New()

// Config for this monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	// ProjectID is the Atlas project ID.
	ProjectID string `yaml:"projectID" validate:"required" `
	// PublicKey is the MongoDB Atlas public API key
	PublicKey string `yaml:"publicKey" validate:"required" `
	// PrivateKey is the MongoDB Atlas private API key
	PrivateKey string `yaml:"privateKey" validate:"required" neverLog:"true"`
	// Timeout for HTTP requests to get MongoDB Atlas process measurements. This should be a duration string that is accepted by https://golang.org/pkg/time/#ParseDuration
	Timeout timeutil.Duration `yaml:"timeout" default:"5s"`
	// EnableCache is a flag to enable caching
	EnableCache bool `yaml:"enableCache" default:"true"`
}

// Monitor for mongodbatlas metrics
type Monitor struct {
	Output             types.FilteringOutput
	ctx                context.Context
	cancel             context.CancelFunc
	client             *mongodbatlas.Client
	projectID          string
	enableCache        bool
	processParamsCache *atomic.Value
	logger             logrus.FieldLogger
}

// Configure monitor
func (m *Monitor) Configure(conf *Config) (err error) {
	m.projectID, m.enableCache, m.processParamsCache = conf.ProjectID, conf.EnableCache, new(atomic.Value)
	m.ctx, m.cancel = context.WithTimeout(context.Background(), conf.Timeout.AsDuration())
	m.client, _ = newClient(conf.PublicKey, conf.PrivateKey)
	interval := time.Duration(conf.IntervalSeconds) * time.Second
	utils.RunOnInterval(m.ctx, func() {
		now := time.Now()
		dps := make([]*datapoint.Datapoint, 0)
		for _, params := range m.GetProcessParams() {
			for _, measurements := range m.GetProcessMeasurements(params) {
				dp := &datapoint.Datapoint{Metric: metricsMap[measurements.Name], Timestamp: now}
				switch measurements.Units {
				case "PERCENT", "MILLISECONDS", "GIGABYTES", "SCALAR":
					dp.MetricType, dp.Value = datapoint.Gauge, newFloatValue(measurements.DataPoints)
				case "BYTES":
					dp.MetricType, dp.Value = datapoint.Gauge, newIntValue(measurements.DataPoints)
				case "BYTES_PER_SECOND", "MEGABYTES_PER_SECOND", "GIGABYTES_PER_HOUR", "SCALAR_PER_SECOND":
					dp.MetricType, dp.Value = datapoint.Rate, newFloatValue(measurements.DataPoints)
				}
				dps = append(dps, dp)
			}
		}
		m.Output.SendDatapoints(dps...)
	}, interval)
	return nil
}

// Shutdown the monitor
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
