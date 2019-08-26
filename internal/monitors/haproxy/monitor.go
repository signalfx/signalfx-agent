package haproxy

import (
	"context"
	"crypto/tls"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	logger "github.com/sirupsen/logrus"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Monitor for Prometheus server metrics Exporter
type Monitor struct {
	Output types.Output
	cancel context.CancelFunc
	ctx    context.Context
}

// Configure the haproxy monitor
func (m *Monitor) Configure(conf *Config) error {
	m.ctx, m.cancel = context.WithCancel(context.Background())
	utils.RunOnInterval(m.ctx, func() {
		dps, err := fetchMetrics(conf.ScrapeURI, conf.Username, conf.Password, conf.SSLVerify, conf.Timeout)
		if err != nil {
			logger.Error(err)
		}
		now := time.Now()
		for _, dp := range dps {
			dp.Timestamp = now
			if conf.UseSignalFxMetricNames && aliases[Metric(dp.Metric)][signalfxName] != "" {
				dp.Metric = aliases[Metric(dp.Metric)][signalfxName]
			}
			m.Output.SendDatapoint(dp)
		}
	}, time.Duration(conf.IntervalSeconds)*time.Second)
	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

func fetchMetrics(uri string, username string, password string, sslVerify bool, timeout time.Duration) ([]*datapoint.Datapoint, error) {
	body, err := reader(uri, username, password, sslVerify, timeout)
	defer func() {
		if body != nil {
			body.Close()
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("can't scrape HAProxy: %v", err)
	}
	csvReader := csv.NewReader(body)
	csvReader.TrailingComma = true
	dps := make([]*datapoint.Datapoint, 0)
	if rows, err := csvReader.ReadAll(); err == nil && len(rows) > 1 {
		// fixing first column header because it is '# pxname' instead of 'pxname'
		rows[0][0] = "pxname"
		for i := 1; i < len(rows); i++ {
			if dpsRow, err := parseRow(rows[0], rows[i]); err == nil {
				dps = append(dps, dpsRow...)
			} else {
				logger.Error(err)
			}
		}
	}
	if len(dps) == 0 {
		return dps, fmt.Errorf("zero datapoints after fetching")
	}
	return dps, nil
}

func reader(uri string, username string, password string, sslVerify bool, timeout time.Duration) (io.ReadCloser, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "http", "https", "file":
		return httpReader(uri, username, password, sslVerify, timeout)
	case "unix":
		return socketReader(u, timeout)
	default:
		return nil, fmt.Errorf("unsupported scheme: %q", u.Scheme)
	}
}

func httpReader(uri string, username string, password string, sslVerify bool, timeout time.Duration) (io.ReadCloser, error) {
	client := http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: !sslVerify}},
	}
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(username, password)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func socketReader(u *url.URL, timeout time.Duration) (io.ReadCloser, error) {
	f, err := net.DialTimeout("unix", u.Path, timeout)
	if err != nil {
		return nil, err
	}
	if err := f.SetDeadline(time.Now().Add(timeout)); err != nil {
		f.Close()
		return nil, err
	}
	cmd := "show stat\n"
	n, err := io.WriteString(f, cmd)
	if err != nil {
		f.Close()
		return nil, err
	}
	if n != len(cmd) {
		f.Close()
		return nil, errors.New("write error")
	}
	return f, nil
}

func parseRow(header []string, row []string) ([]*datapoint.Datapoint, error) {
	const (
		pxnameIndex      = 0
		svnameIndex      = 1
		statusIndex      = 17
		checkStatusIndex = 36
		checkDescIndex   = 65
		modeIndex        = 75
	)
	if len(row) != len(header) {
		return nil, fmt.Errorf("parser expected at least %d csv fields, but got: %d", len(header), len(row))
	}
	dps := make([]*datapoint.Datapoint, 0)
	for index, name := range header {
		if row[index] == "" {
			continue
		}
		var value float64
		switch index {
		// Skipping parsing values of these columns because not numeric.
		case pxnameIndex, svnameIndex, checkStatusIndex, checkDescIndex, modeIndex:
			continue
		case statusIndex:
			value = float64(parseStatusField(row[index]))
		default:
			if i, err := strconv.ParseInt(row[index], 10, 64); err == nil {
				value = float64(i)
			} else {
				logger.Errorf("can't parse csv field value %s: %v", row[index], err)
				continue
			}
		}
		dp := datapoint.New(name, map[string]string{}, datapoint.NewFloatValue(value), metricSet[name].Type, time.Time{})
		if row[pxnameIndex] != "" {
			dp.Dimensions["proxy_name"] = row[pxnameIndex]
		}
		if row[svnameIndex] != "" {
			dp.Dimensions["service_name"] = row[svnameIndex]
		}
		dps = append(dps, dp)
	}
	return dps, nil
}

func parseStatusField(value string) int64 {
	switch value {
	case "UP", "UP 1/3", "UP 2/3", "OPEN", "no check":
		return 1
	case "DOWN", "DOWN 1/2", "NOLB", "MAINT":
		return 0
	}
	return 0
}
