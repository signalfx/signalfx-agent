package atlas

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/signalfx/signalfx-agent/pkg/monitors/mongodb/atlas/measurements"

	"github.com/signalfx/signalfx-agent/pkg/utils"

	"github.com/Sectorbob/mlab-ns2/gae/ns/digest"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"

	"github.com/signalfx/signalfx-agent/pkg/utils/timeutil"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `yaml:",inline"`
	// ProjectID is the Atlas project ID.
	ProjectID string `yaml:"projectID" validate:"required" `
	// PublicKey is the MongoDB Atlas public API key
	PublicKey string `yaml:"publicKey" validate:"required" `
	// PrivateKey is the MongoDB Atlas private API key
	PrivateKey string `yaml:"privateKey" validate:"required" neverLog:"true"`
	// Timeout for HTTP requests to get MongoDB Atlas process measurements. This should be a duration string that is accepted by https://golang.org/pkg/time/#ParseDuration
	Timeout timeutil.Duration `yaml:"timeout" default:"5s"`
	// EnableCache enables locally cached Atlas metric measurements to be used when true. The metric measurements that
	// were supposed to be fetched are in fact always fetched asynchronously and cached.
	EnableCache bool `yaml:"enableCache" default:"true"`
}

// Monitor for MongoDB Atlas metrics
type Monitor struct {
	Output        types.FilteringOutput
	cancel        context.CancelFunc
	processGetter measurements.ProcessesGetter
	diskGetter    measurements.DisksGetter
}

// Configure monitor
func (m *Monitor) Configure(conf *Config) (err error) {
	var client *mongodbatlas.Client
	var processMeasurements measurements.ProcessesMeasurements
	var diskMeasurements measurements.DisksMeasurements

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	timeout := conf.Timeout.AsDuration()

	if client, err = newDigestClient(conf.PublicKey, conf.PrivateKey); err != nil {
		return fmt.Errorf("error making HTTP digest client: %+v", err)
	}

	m.processGetter = measurements.NewProcessesGetter(conf.ProjectID, client, conf.EnableCache)
	m.diskGetter = measurements.NewDisksGetter(conf.ProjectID, client, conf.EnableCache)

	utils.RunOnInterval(ctx, func() {
		processes := m.processGetter.GetProcesses(ctx, timeout)

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			processMeasurements = m.processGetter.GetMeasurements(ctx, timeout, processes)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			diskMeasurements = m.diskGetter.GetMeasurements(ctx, timeout, processes)
		}()

		wg.Wait()

		var dps = make([]*datapoint.Datapoint, 0)

		// Creating metric datapoints from the 1 minute resolution process measurement datapoints
		for k, v := range processMeasurements {
			dps = append(dps, newDps(k, v, "", "")...)
		}

		// Creating metric datapoints from the 1 minute resolution disk measurement datapoints
		for k, v := range diskMeasurements {
			dps = append(dps, newDps(k, v.Measurements, v.DiskName, "")...)
		}

		m.Output.SendDatapoints(dps...)

	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Shutdown the monitor
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

func newDigestClient(publicKey, privateKey string) (*mongodbatlas.Client, error) {
	//Setup a transport to handle digest
	transport := digest.NewTransport(publicKey, privateKey)

	client, err := transport.Client()
	if err != nil {
		return nil, err
	}

	return mongodbatlas.NewClient(client), nil
}

func newDps(process measurements.Process, measurementsArr []*mongodbatlas.Measurements, partition string, database string) []*datapoint.Datapoint {
	var dps = make([]*datapoint.Datapoint, 0)

	for _, measures := range measurementsArr {
		metricValue := newFloatValue(measures.DataPoints)

		if metricValue == nil || metricsMap[measures.Name] == "" {
			continue
		}

		dp := &datapoint.Datapoint{
			Metric:     metricsMap[measures.Name],
			MetricType: datapoint.Gauge,
			Value:      metricValue,
			Dimensions: map[string]string{"host": process.Host, "port": strconv.Itoa(process.Port)},
		}

		if partition != "" {
			dp.Dimensions["partition"] = partition
		}

		if database != "" {
			dp.Dimensions["database"] = database
		}

		dps = append(dps, dp)
	}

	return dps
}

func newFloatValue(dataPoints []*mongodbatlas.DataPoints) datapoint.FloatValue {
	if len(dataPoints) == 0 || dataPoints[0].Value == nil {
		return nil
	}

	return datapoint.NewFloatValue(float64(*dataPoints[0].Value))
}
