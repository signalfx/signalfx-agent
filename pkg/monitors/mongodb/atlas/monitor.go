package atlas

import (
	"context"
	"fmt"
	"strconv"
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
	// EnableCache is a flag to enable caching
	EnableCache bool `yaml:"enableCache" default:"true"`
}

// Monitor for MongoDB Atlas metrics
type Monitor struct {
	Output types.FilteringOutput
	cancel context.CancelFunc
}

// Configure monitor
func (m *Monitor) Configure(conf *Config) (err error) {
	var ctx context.Context
	ctx, m.cancel = context.WithTimeout(context.Background(), conf.Timeout.AsDuration())

	var client *mongodbatlas.Client
	if client, err = newDigestClient(conf.PublicKey, conf.PrivateKey); err != nil {
		return fmt.Errorf("error making HTTP digest client: %+v", err)
	}

	interval := time.Duration(conf.IntervalSeconds) * time.Second

	measurementsGetter := measurements.NewGetter(ctx, client, conf.ProjectID, conf.EnableCache)

	utils.RunOnInterval(ctx, func() {
		now := time.Now()
		dps := make([]*datapoint.Datapoint, 0)
		allMeasurements := measurementsGetter.GetAll()

		// Creating metric datapoints from the 1 minute resolution process measurement datapoints
		for _, process := range allMeasurements.Processes {
			for _, m := range process.Measurements {
				if measurementMetricMap[m.Name] == "" {
					continue
				}
				dps = append(dps, newDp(m.Name, m.DataPoints, now, process.Host, process.Port, "", ""))
			}
		}

		// Creating metric datapoints from the 1 minute resolution disk measurement datapoints
		for _, disk := range allMeasurements.Disks {
			for _, m := range disk.Measurements {
				if measurementMetricMap[m.Name] == "" {
					continue
				}
				dps = append(dps, newDp(m.Name, m.DataPoints, now, disk.Host, disk.Port, disk.PartitionName, ""))
			}
		}

		// Creating metric datapoints from the 1 minute resolution database measurement datapoints
		for _, database := range allMeasurements.Databases {
			for _, m := range database.Measurements {
				if measurementMetricMap[m.Name] == "" {
					continue
				}
				dps = append(dps, newDp(m.Name, m.DataPoints, now, database.Host, database.Port, "", database.DatabaseName))
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

func newDigestClient(publicKey, privateKey string) (*mongodbatlas.Client, error) {
	//Setup a transport to handle digest
	transport := digest.NewTransport(publicKey, privateKey)

	client, err := transport.Client()
	if err != nil {
		return nil, err
	}

	return mongodbatlas.NewClient(client), nil
}

func newDp(measurementName string, dataPoints []*mongodbatlas.DataPoints, timestamp time.Time, host string, port int, partition string, database string) *datapoint.Datapoint {
	dp := &datapoint.Datapoint{
		Metric:     measurementMetricMap[measurementName],
		MetricType: datapoint.Gauge,
		Value:      newFloatValue(dataPoints),
		Timestamp:  timestamp,
		Dimensions: map[string]string{"hostname": host, "port": strconv.Itoa(port)},
	}

	if partition != "" {
		dp.Dimensions["partition"] = partition
	}

	if database != "" {
		dp.Dimensions["database"] = database
	}

	return dp
}

func newFloatValue(dataPoints []*mongodbatlas.DataPoints) datapoint.FloatValue {
	if len(dataPoints) == 0 || dataPoints[0].Value == nil {
		return nil
	}

	return datapoint.NewFloatValue(float64(*dataPoints[0].Value))
}
