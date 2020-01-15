package prometheusexporter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/signalfx/signalfx-agent/pkg/core/common/auth"
	"github.com/signalfx/signalfx-agent/pkg/core/common/httpclient"

	"k8s.io/client-go/rest"

	"github.com/sirupsen/logrus"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
)

func init() {
	RegisterMonitor(monitorMetadata)
}

// Config for this monitor
type Config struct {
	config.MonitorConfig  `yaml:",inline" acceptsEndpoints:"true"`
	httpclient.HTTPConfig `yaml:",inline"`

	// Host of the exporter
	Host string `yaml:"host" validate:"required"`
	// Port of the exporter
	Port uint16 `yaml:"port" validate:"required"`

	// Use pod service account to authenticate.
	UseServiceAccount bool `yaml:"useServiceAccount"`

	// Path to the metrics endpoint on the exporter server, usually `/metrics`
	// (the default).
	MetricPath string `yaml:"metricPath" default:"/metrics"`

	// Send all the metrics that come out of the Prometheus exporter without
	// any filtering.  This option has no effect when using the prometheus
	// exporter monitor directly since there is no built-in filtering, only
	// when embedding it in other monitors.
	SendAllMetrics bool `yaml:"sendAllMetrics"`
}

func (c *Config) GetExtraMetrics() []string {
	// Maintain backwards compatibility with the config flag that existing
	// prior to the new filtering mechanism.
	if c.SendAllMetrics {
		return []string{"*"}
	}
	return nil
}

var _ config.ExtraMetrics = &Config{}
var _ monitors.Collectable = &Monitor{}

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

	monitorName string
	logger      logrus.FieldLogger
	cancel      func()
	fetch       func() (io.ReadCloser, expfmt.Format, error)
}

type fetcher func() (io.ReadCloser, expfmt.Format, error)

// Configure the monitor and kick off volume metric syncing
func (m *Monitor) Configure(conf *Config) error {
	m.logger = logrus.WithFields(logrus.Fields{"monitorType": m.monitorName})

	var bearerToken string

	if conf.UseServiceAccount {
		restConfig, err := rest.InClusterConfig()
		if err != nil {
			return err
		}
		bearerToken = restConfig.BearerToken
		if bearerToken == "" {
			return errors.New("bearer token was empty")
		}
	}

	client, err := conf.HTTPConfig.Build()
	if err != nil {
		return err
	}

	if bearerToken != "" {
		client.Transport = &auth.TransportWithToken{
			RoundTripper: client.Transport,
			Token:        bearerToken,
		}
	}

	url := fmt.Sprintf("%s://%s:%d%s", conf.Scheme(), conf.Host, conf.Port, conf.MetricPath)

	m.fetch = func() (io.ReadCloser, expfmt.Format, error) {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, expfmt.FmtUnknown, err
		}

		resp, err := client.Do(req) // nolint:bodyclose  // We do actually close it after it is returned
		if err != nil {
			return nil, expfmt.FmtUnknown, err
		}

		if resp.StatusCode != 200 {
			body, _ := ioutil.ReadAll(resp.Body)
			return nil, expfmt.FmtUnknown, fmt.Errorf("prometheus exporter at %s returned status %d: %s", url, resp.StatusCode, string(body))
		}

		return resp.Body, expfmt.ResponseFormat(resp.Header), nil
	}

	return nil
}

func (m *Monitor) Collect(ctx context.Context) error {
	dps, err := fetchPrometheusMetrics(m.fetch)
	if err != nil {
		return errors.New("Could not get prometheus metrics")
	}

	now := time.Now()
	for i := range dps {
		dps[i].Timestamp = now
	}
	m.Output.SendDatapoints(dps...)
	return nil
}

func fetchPrometheusMetrics(fetch fetcher) ([]*datapoint.Datapoint, error) {
	metricFamilies, err := doFetch(fetch)
	if err != nil {
		return nil, err
	}

	var dps []*datapoint.Datapoint
	for i := range metricFamilies {
		dps = append(dps, convertMetricFamily(metricFamilies[i])...)
	}
	return dps, nil
}

func doFetch(fetch fetcher) ([]*dto.MetricFamily, error) {
	body, expformat, err := fetch()
	if err != nil {
		return nil, err
	}
	defer body.Close()

	decoder := expfmt.NewDecoder(body, expformat)
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

// RegisterMonitor is a helper for other monitors that simply wrap prometheusexporter.
func RegisterMonitor(meta monitors.Metadata) {
	monitors.Register(&meta, func() interface{} {
		return &Monitor{monitorName: meta.MonitorType}
	},
		&Config{})
}
