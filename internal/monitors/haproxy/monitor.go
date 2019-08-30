package haproxy

import (
	"bufio"
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
	"strings"
	"time"

	"github.com/signalfx/signalfx-agent/internal/utils"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
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
func (m *Monitor) Configure(conf *Config) (err error) {
	m.ctx, m.cancel = context.WithCancel(context.Background())
	utils.RunOnInterval(m.ctx, func() {
		dps, err := getDps(conf)
		if err != nil {
			logger.Error(err)
		}
		now := time.Now()
		for _, dp := range dps {
			dp.Timestamp = now
			if conf.UseSfxNames {
				sfxName := strings.TrimSpace(metricProperties[dp.Metric][sfxNameKey])
				switch sfxName {
				case "":
					logger.Debugf("metric %s has no equivalent SignalFx name", dp.Metric)
				default:
					dp.Metric = sfxName
				}
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

func getDps(conf *Config) ([]*datapoint.Datapoint, error) {
	u, err := url.Parse(conf.ScrapeURI)
	if err != nil {
		return nil, err
	}
	var body io.ReadCloser
	switch u.Scheme {
	case "http", "https", "file":
		if body, err = csvReader(conf); err == nil {
			return getCsvDps(body)
		}
		return nil, fmt.Errorf("can't scrape HAProxy: %v", err)
	case "unix":
		var dps, tmp []*datapoint.Datapoint
		if body, err = cmdReader(u, "show stat\n", conf.Timeout); err == nil {
			if tmp, err = getCsvDps(body); err == nil {
				dps = append(dps, tmp...)
			} else {
				logger.Error(err)
			}
		} else {
			logger.Error(err)
		}
		if body, err = cmdReader(u, "show info\n", conf.Timeout); err == nil {
			if tmp, err = getCmdDps(body); err == nil {
				dps = append(dps, tmp...)
			} else {
				logger.Error(err)
			}
		} else {
			logger.Error(err)
		}
		return dps, nil
	default:
		return nil, fmt.Errorf("unsupported scheme: %q", u.Scheme)
	}
}

func getCsvDps(body io.ReadCloser) ([]*datapoint.Datapoint, error) {
	defer closeBody(body)
	r := csv.NewReader(body)
	r.TrailingComma = true
	var dps []*datapoint.Datapoint
	if rows, err := r.ReadAll(); err == nil && len(rows) > 1 {
		// fixing first column header because it is '# pxname' instead of 'pxname'
		rows[0][0] = "pxname"
		for i := 1; i < len(rows); i++ {
			if row, err := parseRow(rows[0], rows[i]); err == nil {
				dps = append(dps, row...)
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

func getCmdDps(body io.ReadCloser) ([]*datapoint.Datapoint, error) {
	defer closeBody(body)
	sc := bufio.NewScanner(body)
	var dps []*datapoint.Datapoint
	for sc.Scan() {
		subs := strings.SplitN(sc.Text(), ":", 2)
		if len(subs) != 2 {
			logger.Debugf("error splitting string '%s' into 2 substrings using separator ':'", sc.Text())
			continue
		}
		v, err := strconv.ParseInt(strings.TrimSpace(subs[1]), 10, 64)
		if err != nil {
			switch err.(type) {
			case *strconv.NumError:
				logger.Debug(err)
			default:
				logger.Error(err)
			}
			continue
		}
		n := strings.TrimSpace(subs[0])
		dp := datapoint.New(n, map[string]string{}, datapoint.NewFloatValue(float64(v)), metricSet[n].Type, time.Time{})
		dps = append(dps, dp)
	}
	if len(dps) == 0 {
		return dps, fmt.Errorf("zero datapoints after fetching")
	}
	return dps, nil
}

func csvReader(conf *Config) (io.ReadCloser, error) {
	client := http.Client{
		Timeout:   conf.Timeout,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: !conf.SSLVerify}},
	}
	req, err := http.NewRequest("GET", conf.ScrapeURI, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(conf.Username, conf.Password)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		res.Body.Close()
		return nil, fmt.Errorf("HTTP status %d", res.StatusCode)
	}
	return res.Body, nil
}

func cmdReader(u *url.URL, cmd string, timeout time.Duration) (io.ReadCloser, error) {
	f, err := net.DialTimeout("unix", u.Path, timeout)
	if err != nil {
		return nil, err
	}
	if err := f.SetDeadline(time.Now().Add(timeout)); err != nil {
		f.Close()
		return nil, err
	}
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
		pxnameIndex = 0
		svnameIndex = 1
		statusIndex = 17
	)
	if len(row) != len(header) {
		return nil, fmt.Errorf("parser expected at least %d csv fields, but got: %d", len(header), len(row))
	}
	dps := make([]*datapoint.Datapoint, 0)
	for i, name := range header {
		if row[i] == "" {
			continue
		}
		var value float64
		switch i {
		case statusIndex:
			value = float64(parseStatusField(row[i]))
		default:
			v, err := strconv.ParseInt(row[i], 10, 64)
			if err != nil {
				switch err.(type) {
				case *strconv.NumError:
					logger.Debug(err)
				default:
					logger.Error(err)
				}
				continue
			}
			value = float64(v)
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

func parseStatusField(v string) int64 {
	switch v {
	case "UP", "UP 1/3", "UP 2/3", "OPEN", "no check":
		return 1
	case "DOWN", "DOWN 1/2", "NOLB", "MAINT":
		return 0
	}
	return 0
}

func closeBody(body io.ReadCloser) {
	if body != nil {
		body.Close()
	}
}
