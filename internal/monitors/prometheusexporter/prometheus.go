package prometheusexporter

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"k8s.io/client-go/rest"

	"github.com/sirupsen/logrus"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/common/auth"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

func init() {
	RegisterMonitor(monitorMetadata)
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`

	// Host of the exporter
	Host string `yaml:"host" validate:"required"`
	// Port of the exporter
	Port uint16 `yaml:"port" validate:"required"`

	// Basic Auth username to use on each request, if any.
	Username string `yaml:"username"`
	// Basic Auth password to use on each request, if any.
	Password string `yaml:"password" neverLog:"true"`

	// If true, the agent will connect to the exporter using HTTPS instead of plain HTTP.
	UseHTTPS bool `yaml:"useHTTPS"`

	// If useHTTPS is true and this option is also true, the exporter's TLS
	// cert will not be verified.
	SkipVerify bool `yaml:"skipVerify"`
	// Path to the CA cert that has signed the TLS cert, unnecessary
	// if `skipVerify` is set to false.
	CACertPath string `yaml:"caCertPath"`
	// Path to the client TLS cert to use for TLS required connections
	ClientCertPath string `yaml:"clientCertPath"`
	// Path to the client TLS key to use for TLS required connections
	ClientKeyPath string `yaml:"clientKeyPath"`

	// HTTP timeout duration for both read and writes. This should be a
	// duration string that is accepted by https://golang.org/pkg/time/#ParseDuration
	HTTPTimeout time.Duration `yaml:"httpTimeout" default:"10s"`

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
}

type fetcher func() (io.ReadCloser, expfmt.Format, error)

// Configure the monitor and kick off volume metric syncing
func (m *Monitor) Configure(conf *Config) error {
	m.logger = logrus.WithFields(logrus.Fields{"monitorType": m.monitorName})

	var bearerToken string
	tlsConfig := &tls.Config{InsecureSkipVerify: conf.SkipVerify}

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

	tlsConfig, err := auth.TLSConfig(tlsConfig, conf.CACertPath, conf.ClientCertPath, conf.ClientKeyPath)

	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: conf.HTTPTimeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	var scheme string
	if conf.UseHTTPS {
		scheme = "https"
	} else {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s:%d%s", scheme, conf.Host, conf.Port, conf.MetricPath)

	fetch := func() (io.ReadCloser, expfmt.Format, error) {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, expfmt.FmtUnknown, err
		}

		if conf.Username != "" {
			req.SetBasicAuth(conf.Username, conf.Password)
		}

		if bearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+bearerToken)
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

	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())
	utils.RunOnInterval(ctx, func() {
		dps, err := fetchPrometheusMetrics(fetch)
		if err != nil {
			m.logger.WithError(err).Error("Could not get prometheus metrics")
			return
		}

		now := time.Now()
		for i := range dps {
			dps[i].Timestamp = now
		}
		m.Output.SendDatapoints(dps...)
	}, time.Duration(conf.IntervalSeconds)*time.Second)

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
