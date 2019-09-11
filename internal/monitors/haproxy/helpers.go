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

// Map of HAProxy metrics name to their equivalent SignalFx names.
var sfxMetricsMap = map[string]string{
	"conn_tot":           haproxyConnectionTotal,
	"lbtot":              haproxyServerSelectedTotal,
	"bin":                haproxyBytesIn,
	"bout":               haproxyBytesOut,
	"cli_abrt":           haproxyClientAborts,
	"comp_byp":           haproxyCompressBypass,
	"comp_in":            haproxyCompressIn,
	"comp_out":           haproxyCompressOut,
	"comp_rsp":           haproxyCompressResponses,
	"CompressBpsIn":      haproxyCompressBitsPerSecondIn,
	"CompressBpsOut":     haproxyCompressBitsPerSecondOut,
	"CumConns":           haproxyConnections,
	"dreq":               haproxyDeniedRequest,
	"dresp":              haproxyDeniedResponse,
	"downtime":           haproxyDowntime,
	"econ":               haproxyErrorConnections,
	"ereq":               haproxyErrorRequest,
	"eresp":              haproxyErrorResponse,
	"chkfail":            haproxyFailedChecks,
	"wredis":             haproxyRedispatched,
	"req_tot":            haproxyRequestTotal,
	"CumReq":             haproxyRequests,
	"hrsp_1xx":           haproxyResponse1xx,
	"hrsp_2xx":           haproxyResponse2xx,
	"hrsp_3xx":           haproxyResponse3xx,
	"hrsp_4xx":           haproxyResponse4xx,
	"hrsp_5xx":           haproxyResponse5xx,
	"hrsp_other":         haproxyResponseOther,
	"wretr":              haproxyRetries,
	"stot":               haproxySessionTotal,
	"srv_abrt":           haproxyServerAborts,
	"SslCacheLookups":    haproxySslCacheLookups,
	"SslCacheMisses":     haproxySslCacheMisses,
	"CumSslConns":        haproxySslConnections,
	"Uptime_sec":         haproxyUptimeSeconds,
	"act":                haproxyActiveServers,
	"bck":                haproxyBackupServers,
	"check_duration":     haproxyCheckDuration,
	"conn_rate":          haproxyConnectionRate,
	"conn_rate_max":      haproxyConnectionRateMax,
	"CurrConns":          haproxyCurrentConnections,
	"CurrSslConns":       haproxyCurrentSslConnections,
	"dcon":               haproxyDeniedTCPConnections,
	"dses":               haproxyDeniedTCPSessions,
	"Idle_pct":           haproxyIdlePercent,
	"intercepted":        haproxyInterceptedRequests,
	"lastsess":           haproxyLastSession,
	"MaxConnRate":        haproxyMaxConnectionRate,
	"MaxConn":            haproxyMaxConnections,
	"MaxPipes":           haproxyMaxPipes,
	"MaxSessRate":        haproxyMaxSessionRate,
	"MaxSslConns":        haproxyMaxSslConnections,
	"PipesFree":          haproxyPipesFree,
	"PipesUsed":          haproxyPipesUsed,
	"qcur":               haproxyQueueCurrent,
	"qlimit":             haproxyQueueLimit,
	"qmax":               haproxyQueueMax,
	"qtime":              haproxyQueueTimeAverage,
	"req_rate":           haproxyRequestRate,
	"req_rate_max":       haproxyRequestRateMax,
	"rtime":              haproxyResponseTimeAverage,
	"Run_queue":          haproxyRunQueue,
	"scur":               haproxySessionCurrent,
	"rate":               haproxySessionRate,
	"SessRate":           haproxySessionRateAll,
	"rate_lim":           haproxySessionRateLimit,
	"rate_max":           haproxySessionRateMax,
	"ttime":              haproxySessionTimeAverage,
	"SslBackendKeyRate":  haproxySslBackendKeyRate,
	"SslFrontendKeyRate": haproxySslFrontendKeyRate,
	"SslRate":            haproxySslRate,
	"Tasks":              haproxyTasks,
	"throttle":           haproxyThrottle,
	"ZlibMemUsage":       haproxyZlibMemoryUsage,
}

// Creates datapoints by fetching csv stats from an http endpoint.
func newStatsPageDps(conf *Config, proxies map[string]bool) []*datapoint.Datapoint {
	body, err := csvReader(conf)
	if err != nil {
		logger.Errorf("cannot scrape HAProxy: %v", err)
		return nil
	}
	return newStatsDps(body, proxies)
}

// Creates datapoints by running the show stats command.
func newStatsCmdDps(u *url.URL, timeout time.Duration, proxies map[string]bool) []*datapoint.Datapoint {
	body, err := cmdReader(u, "show stat\n", timeout)
	if err != nil {
		logger.Errorf("cannot scrape HAProxy: %v", err)
	}
	return newStatsDps(body, proxies)
}

// Creates datapoints by reading csv stats.
func newStatsDps(body io.ReadCloser, proxiesToMonitor map[string]bool) []*datapoint.Datapoint {
	dps := make([]*datapoint.Datapoint, 0)
	for _, metricValuePairs := range statsMetricValuePairs(body) {
		if len(proxiesToMonitor) != 0 && !proxiesToMonitor[metricValuePairs["pxname"]] && !proxiesToMonitor[metricValuePairs["svname"]] {
			continue
		}
		for metric, value := range metricValuePairs {
			if dp := newDp(sfxMetricsMap[metric], value); dp != nil {
				dp.Dimensions["proxy_name"] = metricValuePairs["pxname"]
				dp.Dimensions["service_name"] = metricValuePairs["svname"]
				// Only stat pid is available here. The related Process_num is unavailable.
				// Decided to increment pid by 1 because it is zero-based while Process_num starts from 1.
				dp.Dimensions["process_num"] = pidPlusPlus(metricValuePairs["pid"])
				dps = append(dps, dp)
			}
		}
	}
	return dps
}

// Creates datapoints from the show info command.
func newInfoDps(u *url.URL, timeout time.Duration) []*datapoint.Datapoint {
	body, err := cmdReader(u, "show info\n", timeout)
	if err != nil {
		logger.Errorf("cannot scrape HAProxy: %v", err)
		return nil
	}
	dps := make([]*datapoint.Datapoint, 0)
	metricValuePairs := infoMetricValuePairs(body)
	for metric, value := range metricValuePairs {
		if dp := newDp(sfxMetricsMap[metric], value); dp != nil {
			dp.Dimensions["process_num"] = metricValuePairs["Process_num"]
			dps = append(dps, dp)
		}
	}
	return dps
}

func statsMetricValuePairs(body io.ReadCloser) map[int]map[string]string {
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

func infoMetricValuePairs(body io.ReadCloser) map[string]string {
	defer closeBody(body)
	sc := bufio.NewScanner(body)
	row := map[string]string{}
	for sc.Scan() {
		s := strings.SplitN(sc.Text(), ":", 2)
		if len(s) != 2 {
			logger.Debugf("could not split string '%s' into 2 substrings using separator ':'", sc.Text())
			continue
		}
		row[strings.TrimSpace(s[0])] = strings.TrimSpace(s[1])
	}
	return row
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

func newDp(metric string, value string) *datapoint.Datapoint {
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

func pidPlusPlus(pid string) string {
	i, err := strconv.Atoi(pid)
	if err != nil {
		logger.Errorf("failed to increment pid by 1")
		return pid
	}
	return strconv.Itoa(i + 1)
}
