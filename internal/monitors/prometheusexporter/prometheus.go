package prometheusexporter

import (
	"context"
	"crypto/tls"
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

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`

	// Host of the exporter
	Host string `yaml:"host" validate:"required"`
	// Port of the exporter
	Port uint16 `yaml:"port" validate:"required"`

	// If true, the agent will connect to the exporter using HTTPS instead of
	// plain HTTP.
	UseHTTPS bool `yaml:"useHTTPS"`
	// If useHTTPS is true and this option is also true, the exporter's TLS
	// cert will not be verified.
	SkipVerify bool `yaml:"skipVerify"`

	// Path to the metrics endpoint on the exporter server, usually `/metrics`
	// (the default).
	MetricPath string `yaml:"metricPath" default:"/metrics"`

	// Send all the metrics that come out of the Prometheus exporter without
	// any filtering.  This option has no effect when using the prometheus
	// exporter monitor directly since there is no built-in filtering, only
	// when embedding it in other monitors.
	SendAllMetrics bool `yaml:"sendAllMetrics"`
}

// Monitor for prometheus exporter metrics
type Monitor struct {
	Output types.Output
	// Optional set of metric names that will be sent by default, all other
	// metrics derived from the exporter being dropped.
	IncludedMetrics map[string]bool
	// Extra dimensions to add in addition to those specified in the config.
	ExtraDimensions map[string]string
	// If true, IncludedMetrics is ignored and everything is sent.
	SendAll bool
	cancel  func()
	client  *http.Client
}

type filteringOutput struct {
	types.Output
	includedMetrics map[string]bool
}

var _ types.Output = &filteringOutput{}

func (fo *filteringOutput) SendDatapoint(dp *datapoint.Datapoint) {
	if !fo.includedMetrics[dp.Metric] {
		return
	}
	fo.Output.SendDatapoint(dp)
}

// Configure the monitor and kick off volume metric syncing
func (m *Monitor) Configure(conf *Config) error {
	// This is a temporary hack until the generic metric filtering/grouping
	// work is done.  This should be removable once that is done and the logic
	// lives in the core Output instance.
	if m.IncludedMetrics != nil && !conf.SendAllMetrics {
		m.Output = &filteringOutput{Output: m.Output, includedMetrics: m.IncludedMetrics}
	}

	m.client = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: conf.SkipVerify},
		},
	}

	var scheme string
	if conf.UseHTTPS {
		scheme = "https"
	} else {
		scheme = "http"
	}

	host := conf.Host
	// Handle IPv6 addresses properly
	if strings.ContainsAny(host, ":") {
		host = "[" + host + "]"
	}
	url := fmt.Sprintf("%s://%s:%d%s", scheme, host, conf.Port, conf.MetricPath)

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
	defer resp.Body.Close()

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
