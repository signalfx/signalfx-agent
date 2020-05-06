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
	"strconv"
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
	config.MonitorConfig  `yaml:",inline" singleInstance:"false" acceptsEndpoints:"true"`
	httpclient.HTTPConfig `yaml:",inline"`
	// Optional HTTP request body as string like '{"foo":"bar"}'
	RequestBody string `yaml:"requestBody"`
	// Do not follow redirect.
	NoRedirects bool `yaml:"noRedirects" default:"false"`
	// HTTP request method to use.
	Method string `yaml:"method" default:"GET"`
	// List of HTTP URLs to monitor.
	URLs []string `yaml:"urls" validate:"required"`
	// Optional Regex to match on URL(s) response(s).
	Regex string `yaml:"regex"`
	// Desired code to match for URL(s) response(s).
	DesiredCode int `yaml:"desiredCode" default:"200"`
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
}

// Configure and kick off internal metric collection
func (m *Monitor) Configure(conf *Config) (err error) {
	m.conf = conf
	m.logger = logrus.WithFields(logrus.Fields{"monitorType": m.monitorName})
	// always try https if available
	m.conf.UseHTTPS = true
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
	utils.RunOnInterval(ctx, func() {
		// get stats for each website
		for _, site := range m.conf.URLs {
			logger := m.logger.WithFields(logrus.Fields{"url": site})
			_, err := url.Parse(site)
			if err != nil {
				logger.WithError(err).Error("could not parse url, ignore this url")
				continue
			}
			dps, lastURL, err := m.getHTTPStats(site, logger)
			if err == nil {
				parsedURL, _ := url.Parse(lastURL)
				if parsedURL.Scheme == "https" {
					tlsDps, err := m.getTLSStats(parsedURL, logger)
					if err == nil {
						dps = append(dps, tlsDps...)
					} else {
						logger.WithError(err).Error("Failed gathering TLS stats")
					}
				}
			} else {
				logger.WithError(err).Error("Failed gathering HTTP stats, ignore other stats")
			}
			// for _, v := range dps {
			// 	logger.Warn(v.Metric, v.Value, v.Dimensions)
			// }
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

func (m *Monitor) getTLSStats(site *url.URL, logger *logrus.Entry) (dps []*datapoint.Datapoint, err error) {
	// use as an fmt.Stringer
	host := site.Hostname()
	port := site.Port()
	valid := true
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
		valid = false
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
			valid = false
		}
	}
	dimensions := map[string]string{
		"url":        site.String(),
		"serverName": host,
		"isValid":    strconv.FormatBool(valid),
	}
	dps = append(dps, datapoint.New(httpCertExpiry, dimensions, datapoint.NewFloatValue(secondsLeft), datapoint.Gauge, time.Time{}))
	return dps, nil
}

func (m *Monitor) getHTTPStats(site string, logger *logrus.Entry) (dps []*datapoint.Datapoint, lastURL string, err error) {
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
	req, err := http.NewRequest(m.conf.Method, site, body)
	if err != nil {
		return
	}
	// starts timer
	now := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	responseTime := time.Since(now).Seconds()
	defer resp.Body.Close()
	lastURL = resp.Request.URL.String()
	dimensions := map[string]string{
		"url":    lastURL,
		"method": m.conf.Method,
	}
	matchCode := false
	statusCode := resp.StatusCode
	if statusCode == m.conf.DesiredCode {
		matchCode = true
	}
	dps = append(dps,
		datapoint.New(httpResponseTime, dimensions, datapoint.NewFloatValue(responseTime), datapoint.Gauge, time.Time{}),
		datapoint.New(httpStatusCode, addDimensions(dimensions, map[string]string{
			"matchCode":   strconv.FormatBool(matchCode),
			"desiredCode": strconv.Itoa(m.conf.DesiredCode),
		}), datapoint.NewIntValue(int64(statusCode)), datapoint.Gauge, time.Time{}),
	)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.WithError(err).Error("could not parse body response")
	} else {
		contentDimensions := dimensions
		if m.conf.Regex != "" {
			matchRegex := false
			if m.regex.Match(bodyBytes) {
				matchRegex = true
			}
			contentDimensions = addDimensions(contentDimensions, map[string]string{
				"matchRegex": strconv.FormatBool(matchRegex),
			})
		}
		dps = append(dps, datapoint.New(httpContentLength, contentDimensions, datapoint.NewIntValue(int64(len(bodyBytes))), datapoint.Gauge, time.Time{}))
	}
	return dps, lastURL, nil
}

func addDimensions(globalDimensions map[string]string, newDimensions map[string]string) (dimensions map[string]string) {
	dimensions = make(map[string]string)
	for k, v := range globalDimensions {
		dimensions[k] = v
	}
	for k, v := range newDimensions {
		dimensions[k] = v
	}
	return dimensions
}
