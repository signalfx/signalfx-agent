package prometheus

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

const monitorType = "prometheus-exporter"

// MONITOR(prometheus-exporter): This monitor reads metrics from a [Prometheus
// exporter](https://prometheus.io/docs/instrumenting/exporters/) endpoint.
//
// All metric types *except for* histograms are supported.  Histograms are
// ignored. All Prometheus labels will be converted directly to SignalFx
// dimensions.
//
// This supports service discovery so you can set a discovery rule like: `port
// >= 9100 && port <= 9500 && containerImage =~ "exporter"`, assuming you are
// running exporters in container images that have the word "exporter" in them
// and fall within the standard exporter port range.  In K8s, you could also
// try matching on the container port name as defined in the pod spec, which is
// the `name` variable in discovery rules for the `k8s-api` observer.
//
// Filtering can be very useful here since exporters tend to be fairly verbose.

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`

	// Host of the exporter
	Host string `yaml:"host" validate:"required"`
	// Port of the exporter
	Port uint16 `yaml:"port" validate:"required"`
	Name string `yaml:"name"`

	// Path to the metrics endpoint on the exporter server, usually `/metrics`
	// (the default).`
	MetricPath string `yaml:"metricPath" default:"/metrics"`
}

// Monitor for prometheus exporter metrics
type Monitor struct {
	Output types.Output
	cancel func()
	client *http.Client
}

// Configure the monitor and kick off volume metric syncing
func (m *Monitor) Configure(conf *Config) error {
	m.client = &http.Client{
		Timeout: 10 * time.Second,
	}

	host := conf.Host
	// Handle IPv6 addresses properly
	if strings.ContainsAny(host, ":") {
		host = "[" + host + "]"
	}
	url := fmt.Sprintf("http://%s:%d%s", host, conf.Port, conf.MetricPath)

	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())
	utils.RunOnInterval(ctx, func() {
		dps, err := fetchPrometheusMetrics(m.client, url)
		if err != nil {
			logger.WithError(err).Error("Could not get prometheus metrics")
			return
		}

		now := time.Now()
		for i := range dps {
			dps[i].Timestamp = now
			m.Output.SendDatapoint(dps[i])
		}
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

func fetchPrometheusMetrics(client *http.Client, url string) ([]*datapoint.Datapoint, error) {
	metricFamilies, err := doFetch(client, url)
	if err != nil {
		return nil, err
	}

	var dps []*datapoint.Datapoint
	for i := range metricFamilies {
		dps = append(dps, convertMetricFamily(metricFamilies[i])...)
	}
	return dps, nil
}

func doFetch(client *http.Client, url string) ([]*dto.MetricFamily, error) {
	// Prometheus 2.0 deprecated protobuf and now only does the text format.
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Prometheus exporter at %s returned status %d", url, resp.StatusCode)
	}

	decoder := expfmt.NewDecoder(resp.Body, expfmt.ResponseFormat(resp.Header))
	var mfs []*dto.MetricFamily

	for {
		var mf dto.MetricFamily
		err := decoder.Decode(&mf)

		if err == io.EOF {
			return mfs, nil
		} else if err != nil {
			return nil, err
		}

		mfs = append(mfs, &mf)
	}
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
