package haproxy

import (
	"bufio"
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

	"github.com/signalfx/golib/datapoint"
	logger "github.com/sirupsen/logrus"
)

func newStatPageDatapoints(body io.ReadCloser, sfxMetricNames map[string]string, proxiesToMonitor map[string]bool) []*datapoint.Datapoint {
	dps := make([]*datapoint.Datapoint, 0)
	for _, metricValuePairs := range statsPageMetricValuePairs(body) {
		if len(proxiesToMonitor) != 0 && !proxiesToMonitor[metricValuePairs["pxname"]] && !proxiesToMonitor[metricValuePairs["svname"]] {
			continue
		}
		for metric, value := range metricValuePairs {
			if dp := newDatapoint(sfxMetricNames[metric], value); dp != nil {
				dp.Dimensions["proxy_name"] = metricValuePairs["pxname"]
				dp.Dimensions["service_name"] = metricValuePairs["svname"]
				dp.Dimensions["process_num"] = metricValuePairs["pid"]
				dps = append(dps, dp)
			}
		}
	}
	return dps
}

func newShowStatCommandDatapoints(body io.ReadCloser, sfxMetricNames map[string]string, proxiesToMonitor map[string]bool) []*datapoint.Datapoint {
	return newStatPageDatapoints(body, sfxMetricNames, proxiesToMonitor)
}

func newShowInfoCommandDatapoints(body io.ReadCloser, sfxMetricNames map[string]string) []*datapoint.Datapoint {
	dps := make([]*datapoint.Datapoint, 0)
	for _, metricValuePairs := range showInfoCommandMetricValuePairs(body) {
		for metric, value := range metricValuePairs {
			if dp := newDatapoint(sfxMetricNames[metric], value); dp != nil {
				dp.Dimensions["process_num"] = metricValuePairs["Process_num"]
				dps = append(dps, dp)
			}
		}
	}
	return dps
}

func statsPageMetricValuePairs(body io.ReadCloser) map[int]map[string]string /*([]*datapoint.Datapoint, error)*/ {
	defer closeBody(body)
	r := csv.NewReader(body)
	r.TrimLeadingSpace = true
	r.TrailingComma = true
	rows := map[int]map[string]string{}
	if table, err := r.ReadAll(); err == nil && len(table) > 1 {
		// fixing first column header because it is '# pxname' instead of 'pxname'
		table[0][0] = "pxname"
		for rowIndex := 1; rowIndex < len(table); rowIndex++ {
			if rows[rowIndex-1] == nil {
				rows[rowIndex-1] = map[string]string{}
			}
			for j, colName := range table[0] {
				rows[rowIndex-1][colName] = table[rowIndex][j]
			}
		}
	}
	return rows
}

func showInfoCommandMetricValuePairs(body io.ReadCloser) map[int]map[string]string /*([]*datapoint.Datapoint, error)*/ {
	defer closeBody(body)
	sc := bufio.NewScanner(body)
	rows := map[int]map[string]string{}
	for sc.Scan() {
		s := strings.SplitN(sc.Text(), ":", 2)
		if len(s) != 2 {
			logger.Debugf("could not split string '%s' into 2 substrings using separator ':'", sc.Text())
			continue
		}
		if rows[0] == nil {
			rows[0] = map[string]string{}
		}
		rows[0][strings.TrimSpace(s[0])] = strings.TrimSpace(s[1])
	}
	return rows
}

func csvReader(conf *Config) (io.ReadCloser, error) {
	client := http.Client{
		Timeout:   conf.Timeout,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: !conf.SSLVerify}},
	}
	req, err := http.NewRequest("GET", conf.URL, nil)
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

func commandReader(u *url.URL, cmd string, timeout time.Duration) (io.ReadCloser, error) {
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

func newDatapoint(metric string, value string) *datapoint.Datapoint {
	if metric == "" || value == "" {
		return nil
	}
	dp := datapoint.New(metric, map[string]string{}, nil, metricSet[metric].Type, time.Time{})
	switch metric {
	case "status":
		dp.Value = datapoint.NewFloatValue(float64(parseStatusField(value)))
	default:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			switch err.(type) {
			case *strconv.NumError:
				logger.Debug(err)
			default:
				logger.Error(err)
			}
			return nil
		}
		dp.Value = datapoint.NewFloatValue(float64(v))
	}
	return dp
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
