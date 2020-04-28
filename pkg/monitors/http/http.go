package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	"github.com/sirupsen/logrus"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `singleInstance:"false" acceptsEndpoints:"false"`
	Body                 string            `yaml:"body"`
	FollowRedirects      bool              `yaml:"followRedirects" default:"true"`
	Headers              map[string]string `yaml:"headers"`
	Method               string            `yaml:"method" default:"GET"`
	Timeout              int               `yaml:"timeout" default:"5"`
	URLs                 []string          `yaml:"urls"`
	Regex                string            `yaml:"regex"`
}

// Monitor that collect metrics
type Monitor struct {
	Output types.FilteringOutput
	cancel func()
	logger logrus.FieldLogger
}

// Configure and kick off internal metric collection
func (m *Monitor) Configure(conf *Config) error {
	m.logger = logrus.WithFields(logrus.Fields{"monitorType": monitorType})
	// Start the metric gathering process here
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())
	utils.RunOnInterval(ctx, func() {
		tlsCfg := &tls.Config{InsecureSkipVerify: true}
		timeoutDuration := time.Duration(conf.Timeout) * time.Second
		// add data
		var body io.Reader
		if conf.Body != "" {
			body = strings.NewReader(conf.Body)
		}
		// get stats for each website
		for _, site := range conf.URLs {
			m.logger = m.logger.WithFields(logrus.Fields{"url": site})
			m.logger.Debug("starting monitor url")
			var dps []*datapoint.Datapoint
			dimensions := map[string]string{"url": site}
			dialer := &net.Dialer{Timeout: timeoutDuration}
			client := &http.Client{
				Transport: &http.Transport{
					DialContext:       dialer.DialContext,
					DisableKeepAlives: true,
					TLSClientConfig:   tlsCfg,
				},
				Timeout: timeoutDuration,
			}
			if !conf.FollowRedirects {
				client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				}
			}
			req, err := http.NewRequest(conf.Method, site, body)
			if err != nil {
				m.logger.WithError(err).Error("could not create new request")
				continue
			}
			// add hearders
			for key, val := range conf.Headers {
				req.Header.Add(key, val)
				if key == "Host" {
					req.Host = val
				}
			}
			// starts timer
			now := time.Now()
			resp, err := client.Do(req)
			if err != nil {
				m.logger.WithError(err).Error("could not do the request")
				continue
			}
			dps = append(dps, datapoint.New("http.response_time", dimensions, datapoint.NewFloatValue(time.Since(now).Seconds()), datapoint.Gauge, time.Time{}))
			dps = append(dps, datapoint.New("http.status_code", dimensions, datapoint.NewIntValue(int64(resp.StatusCode)), datapoint.Gauge, time.Time{}))
			defer resp.Body.Close()
			bodyBytes, err := ioutil.ReadAll(resp.Body)
			dps = append(dps, datapoint.New("http.content_length", dimensions, datapoint.NewIntValue(int64(len(bodyBytes))), datapoint.Gauge, time.Time{}))
			if err != nil {
				m.logger.WithError(err).Error("could not parse body")
			} else if conf.Regex != "" {
				var match int64 = 0
				regex, err := regexp.Compile(conf.Regex)
				if err != nil {
					m.logger.WithError(err).Error("failed to compile regular expression")
				}
				if regex.Match(bodyBytes) {
					match = 1
				}
				dps = append(dps, datapoint.New("http.regex_match", dimensions, datapoint.NewIntValue(match), datapoint.Gauge, time.Time{}))
			}
			parsedURL, err := url.Parse(resp.Request.URL.String())
			if err != nil {
				m.logger.WithError(err).Error("could not parse url")
				m.Output.SendDatapoints(dps...)
				continue
			}
			// check TLS if last url followed is https
			if err == nil && parsedURL.Scheme == "https" {
				host := parsedURL.Hostname()
				port := parsedURL.Port()
				tlsCfg.ServerName = host
				if port == "" {
					port = "443"
				}
				conn, err := tls.DialWithDialer(dialer, "tcp", host+":"+port, tlsCfg)
				if err != nil {
					m.logger.WithError(err).Error("could not connect to server")
					m.Output.SendDatapoints(dps...)
					continue
				}
				defer conn.Close()
				err = conn.Handshake()
				if err != nil {
					m.logger.WithError(err).Error("failed during handshake")
				}
				var valid int64 = 1
				certs := conn.ConnectionState().PeerCertificates
				for i, cert := range certs {
					opts := x509.VerifyOptions{
						Intermediates: x509.NewCertPool(),
					}
					if i == 0 {
						opts.DNSName = host
						for j, cert := range certs {
							if j != 0 {
								opts.Intermediates.AddCert(cert)
							}
						}
						dps = append(dps, datapoint.New("http.certificate_expiration", dimensions, datapoint.NewFloatValue(cert.NotAfter.Sub(now).Seconds()), datapoint.Gauge, time.Time{}))
					}
					_, err := cert.Verify(opts)
					if err != nil {
						valid = 0
						m.logger.WithError(err).Info("failed verify certificate")
					}
					dps = append(dps, datapoint.New("http.certificate_valid", dimensions, datapoint.NewIntValue(valid), datapoint.Gauge, time.Time{}))
				}
			}
			m.Output.SendDatapoints(dps...)
		}
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Shutdown the monitor
func (m *Monitor) Shutdown() {
	// Stop any long-running go routines here
	if m.cancel != nil {
		m.cancel()
	}
}
