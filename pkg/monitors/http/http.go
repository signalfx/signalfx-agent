package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/core/common/auth"
	"github.com/signalfx/signalfx-agent/pkg/core/common/httpclient"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	"github.com/sirupsen/logrus"
)

func init() {

	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{monitorName: monitorMetadata.MonitorType} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" singleInstance:"false" acceptsEndpoints:"true"`
	// Host/IP to monitor
	Host string `yaml:"host"`
	// Port of the HTTP server to monitor
	Port uint16 `yaml:"port"`
	// HTTP path to use in the test request
	Path string `yaml:"path"`

	httpclient.HTTPConfig `yaml:",inline"`
	// Optional HTTP request body as string like '{"foo":"bar"}'
	RequestBody string `yaml:"requestBody"`
	// Do not follow redirect.
	NoRedirects bool `yaml:"noRedirects" default:"false"`
	// HTTP request method to use.
	Method string `yaml:"method" default:"GET"`
	// DEPRECATED: list of HTTP URLs to monitor. Use `host`/`port`/`useHTTPS`/`path` instead.
	URLs []string `yaml:"urls"`
	// Optional Regex to match on URL(s) response(s).
	Regex string `yaml:"regex"`
	// Desired code to match for URL(s) response(s).
	DesiredCode int `yaml:"desiredCode" default:"200"`
	// Add `last_url` dimension which could differ from `url` when redirection is followed.
	AddLastURL bool `yaml:"addLastURL" default:"false"`
}

// Monitor that collect metrics
type Monitor struct {
	Output types.FilteringOutput
	cancel context.CancelFunc
	//ctx         context.Context
	logger      logrus.FieldLogger
	conf        *Config
	monitorName string
	regex       *regexp.Regexp
	URLs        []*url.URL
}

// Configure and kick off internal metric collection
func (m *Monitor) Configure(conf *Config) (err error) {
	m.conf = conf
	m.logger = logrus.WithFields(logrus.Fields{"monitorType": m.monitorName})
	// Ignore certificate error which will be checked after
	m.conf.SkipVerify = true

	if m.conf.Regex != "" {
		// Compile regex
		m.regex, err = regexp.Compile(m.conf.Regex)
		if err != nil {
			m.logger.WithError(err).Error("failed to compile regular expression")
		}
	}

	// Start the metric gathering process here
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	if m.conf.Host != "" {
		if m.conf.Port == 0 {
			m.conf.Port = m.conf.DefaultPort()
		}
		clientURL, err := m.forgeURL(fmt.Sprintf("%s://%s:%d%s", m.conf.Scheme(), m.conf.Host, m.conf.Port, m.conf.Path))
		if err != nil {
			m.logger.WithError(err).Error("error configuring url from http client, ignore it")
		} else {
			m.URLs = append(m.URLs, clientURL)
		}
	} else {
		// always try https if available. This is for backwards compat.
		m.conf.UseHTTPS = true
	}
	for _, site := range m.conf.URLs {
		stringURL, err := m.forgeURL(site)
		if err != nil {
			m.logger.WithField("url", site).WithError(err).Error("error configuring url from list, ignore it")
			continue
		}
		m.URLs = append(m.URLs, stringURL)
	}

	utils.RunOnInterval(ctx, func() {
		// get stats for each website
		for _, site := range m.URLs {
			logger := m.logger.WithFields(logrus.Fields{"url": site.String()})

			dps, lastURL, err := m.getHTTPStats(site, logger)
			if err == nil {
				if lastURL.Scheme == "https" {
					tlsDps, err := m.getTLSStats(lastURL, logger)
					if err == nil {
						dps = append(dps, tlsDps...)
					} else {
						logger.WithError(err).Error("Failed gathering TLS stats")
					}
				}
			} else {
				logger.WithError(err).Error("Failed gathering HTTP stats, ignore other stats")
			}

			for i := range dps {
				dps[i].Dimensions["url"] = site.String()
			}

			if m.conf.AddLastURL && !m.conf.NoRedirects {
				normalizedURL, _ := m.forgeURL(fmt.Sprintf("%s://%s:%s%s", lastURL.Scheme, lastURL.Hostname(), lastURL.Port(), lastURL.Path))
				if site.String() != normalizedURL.String() {
					logger.WithField("last_url", normalizedURL.String()).Debug("URL redirected")
					for i := range dps {
						dps[i].Dimensions["last_url"] = normalizedURL.String()
					}
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

func (m *Monitor) forgeURL(site string) (normalizedURL *url.URL, err error) {
	stringURL, err := url.Parse(site)
	if err != nil {
		return
	}
	port := stringURL.Port()
	if port == "" {
		port = "80"
		if stringURL.Scheme == "https" {
			port = "443"
		}
	}
	path := stringURL.Path
	if path == "" {
		path = "/"
	}
	normalizedURL, err = url.Parse(fmt.Sprintf("%s://%s:%s%s", stringURL.Scheme, stringURL.Hostname(), port, path))
	if err != nil {
		return
	}
	return
}

func (m *Monitor) getTLSStats(site *url.URL, logger *logrus.Entry) (dps []*datapoint.Datapoint, err error) {
	// use as an fmt.Stringer
	host := site.Hostname()
	port := site.Port()

	var valid int64 = 1
	var secondsLeft float64

	if port == "" {
		port = "443"
	}

	ipConn, err := net.Dial("tcp", host+":"+port)
	if err != nil {
		return
	}
	defer ipConn.Close()

	tlsCfg := &tls.Config{
		InsecureSkipVerify: m.conf.SkipVerify,
		ServerName:         host,
	}

	if _, err := auth.TLSConfig(tlsCfg, m.conf.CACertPath, m.conf.ClientCertPath, m.conf.ClientKeyPath); err != nil {
		return nil, err
	}

	conn := tls.Client(ipConn, tlsCfg)
	if err != nil {
		return
	}
	defer conn.Close()

	err = conn.Handshake()
	if err != nil {
		logger.WithError(err).Error("failed during handshake")
		valid = 0
	}

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
			secondsLeft = time.Until(cert.NotAfter).Seconds()
		}
		_, err := cert.Verify(opts)
		if err != nil {
			logger.WithError(err).Info("failed verify certificate")
			valid = 0
		}
	}

	dimensions := map[string]string{
		"server_name": host,
	}

	dps = append(dps,
		datapoint.New(httpCertExpiry, dimensions, datapoint.NewFloatValue(secondsLeft), datapoint.Gauge, time.Time{}),
		datapoint.New(httpCertValid, dimensions, datapoint.NewIntValue(valid), datapoint.Gauge, time.Time{}))

	return dps, nil
}

func (m *Monitor) getHTTPStats(site fmt.Stringer, logger *logrus.Entry) (dps []*datapoint.Datapoint, lastURL *url.URL, err error) {
	// do not suggest fmt.Stringer
	// Init http client
	client, err := m.conf.HTTPConfig.Build()
	if err != nil {
		return
	}

	// Init body if applicable
	var body io.Reader
	if m.conf.RequestBody != "" {
		body = strings.NewReader(m.conf.RequestBody)
	}

	if m.conf.NoRedirects {
		logger.Debug("Do not follow redirects")
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	req, err := http.NewRequest(m.conf.Method, site.String(), body)
	if err != nil {
		return
	}

	// starts timer
	now := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	responseTime := time.Since(now).Seconds()

	lastURL = resp.Request.URL

	dimensions := map[string]string{
		"method": m.conf.Method,
	}

	statusCode := int64(resp.StatusCode)

	var matchCode int64 = 0
	if statusCode == int64(m.conf.DesiredCode) {
		matchCode = 1
	}

	dps = append(dps,
		datapoint.New(httpResponseTime, dimensions, datapoint.NewFloatValue(responseTime), datapoint.Gauge, time.Time{}),
		datapoint.New(httpStatusCode, dimensions, datapoint.NewIntValue(statusCode), datapoint.Gauge, time.Time{}),
		datapoint.New(httpCodeMatched, dimensions, datapoint.NewIntValue(matchCode), datapoint.Gauge, time.Time{}),
	)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.WithError(err).Error("could not parse body response")
	} else {
		dps = append(dps, datapoint.New(httpContentLength, dimensions, datapoint.NewIntValue(int64(len(bodyBytes))), datapoint.Gauge, time.Time{}))

		if m.conf.Regex != "" {
			var matchRegex int64 = 0
			if m.regex.Match(bodyBytes) {
				matchRegex = 1
			}
			dps = append(dps, datapoint.New(httpRegexMatched, dimensions, datapoint.NewIntValue(matchRegex), datapoint.Gauge, time.Time{}))
		}
	}
	return dps, lastURL, err
}
