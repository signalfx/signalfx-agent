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
	"conn_tot":           counterConnectionTotal,
	"lbtot":              counterServerSelectedTotal,
	"bin":                deriveBytesIn,
	"bout":               deriveBytesOut,
	"cli_abrt":           deriveCliAbrt,
	"comp_byp":           deriveCompByp,
	"comp_in":            deriveCompIn,
	"comp_out":           deriveCompOut,
	"comp_rsp":           deriveCompRsp,
	"CompressBpsIn":      deriveCompressBpsIn,
	"CompressBpsOut":     deriveCompressBpsOut,
	"CumConns":           deriveConnections,
	"dreq":               deriveDeniedRequest,
	"dresp":              deriveDeniedResponse,
	"downtime":           deriveDowntime,
	"econ":               deriveErrorConnectiont,
	"ereq":               deriveErrorRequest,
	"eresp":              deriveErrorResponse,
	"chkfail":            deriveFailedChecks,
	"wredis":             deriveRedispatched,
	"req_tot":            deriveRequestTotal,
	"CumReq":             deriveRequests,
	"hrsp_1xx":           deriveResponse1xx,
	"hrsp_2xx":           deriveResponse2xx,
	"hrsp_3xx":           deriveResponse3xx,
	"hrsp_4xx":           deriveResponse4xx,
	"hrsp_5xx":           deriveResponse5xx,
	"hrsp_other":         deriveResponseOther,
	"wretr":              deriveRetries,
	"stot":               deriveSessionTotal,
	"srv_abrt":           deriveSrvAbrt,
	"SslCacheLookups":    deriveSslCacheLookups,
	"SslCacheMisses":     deriveSslCacheMisses,
	"CumSslConns":        deriveSslConnections,
	"Uptime_sec":         deriveUptimeSeconds,
	"act":                gaugeActiveServers,
	"bck":                gaugeBackupServers,
	"check_duration":     gaugeCheckDuration,
	"conn_rate":          gaugeConnectionRate,
	"conn_rate_max":      gaugeConnectionRateMax,
	"CurrConns":          gaugeCurrentConnections,
	"CurrSslConns":       gaugeCurrentSslConnections,
	"dcon":               gaugeDeniedTCPConnections,
	"dses":               gaugeDeniedTCPSessions,
	"Idle_pct":           gaugeIdlePct,
	"intercepted":        gaugeInterceptedRequests,
	"lastsess":           gaugeLastSession,
	"MaxConnRate":        gaugeMaxConnectionRate,
	"MaxConn":            gaugeMaxConnections,
	"MaxPipes":           gaugeMaxPipes,
	"MaxSessRate":        gaugeMaxSessionRate,
	"MaxSslConns":        gaugeMaxSslConnections,
	"PipesFree":          gaugePipesFree,
	"PipesUsed":          gaugePipesUsed,
	"qcur":               gaugeQueueCurrent,
	"qlimit":             gaugeQueueLimit,
	"qmax":               gaugeQueueMax,
	"qtime":              gaugeQueueTimeAvg,
	"req_rate":           gaugeRequestRate,
	"req_rate_max":       gaugeRequestRateMax,
	"rtime":              gaugeResponseTimeAvg,
	"Run_queue":          gaugeRunQueue,
	"scur":               gaugeSessionCurrent,
	"rate":               gaugeSessionRate,
	"SessRate":           gaugeSessionRateAll,
	"rate_lim":           gaugeSessionRateLimit,
	"rate_max":           gaugeSessionRateMax,
	"ttime":              gaugeSessionTimeAverage,
	"SslBackendKeyRate":  gaugeSslBackendKeyRate,
	"SslFrontendKeyRate": gaugeSslFrontendKeyRate,
	"SslRate":            gaugeSslRate,
	"Tasks":              gaugeTasks,
	"throttle":           gaugeThrottle,
	"ZlibMemUsage":       gaugeZlibMemUsage,
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
