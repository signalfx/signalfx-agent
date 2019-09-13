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
	"sync"
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

const (
	repeatFetchAfter = 1000 * time.Millisecond
	maxRoutines      = 16
)

type dpsChans struct {
	chans map[string]chan []*datapoint.Datapoint
	lock  sync.Mutex
}

func (d *dpsChans) getChan(k string) chan []*datapoint.Datapoint {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.chans == nil {
		d.chans = make(map[string]chan []*datapoint.Datapoint)
	}
	if d.chans[k] == nil {
		d.chans[k] = make(chan []*datapoint.Datapoint, 1)
	}
	return d.chans[k]
}

// Creates datapoints for all HAProxy processes from csv stats fetched through http.
func (m *Monitor) fetchAllHTTP(ctx context.Context, conf *Config, numReportedPids int) []*datapoint.Datapoint {
	var statsDpsChans dpsChans
	var wg sync.WaitGroup
	for i := 0; i < min1(maxRoutines, numReportedPids); i++ {
		wg.Add(1)
		go fetchRepeatStats(ctx, m.fetchHTTP, conf, numReportedPids, &wg, &statsDpsChans)
	}
	wg.Wait()
	dps := make([]*datapoint.Datapoint, 0)
	for _, v := range statsDpsChans.chans {
		dps = append(dps, <-v...)
	}
	return dps
}

// Writes into channel, csv stats datapoints.
func fetchRepeatStats(ctx context.Context, fn func(conf *Config) ([]*datapoint.Datapoint, error), conf *Config, numReportedPids int, wg *sync.WaitGroup, statsChans *dpsChans) {
	timer := time.NewTicker(repeatFetchAfter)
	defer timer.Stop()
	for range timer.C {
		select {
		case <-ctx.Done():
			logger.Errorf("failed to write 'show stats' datapoints to channel: %+v", ctx.Err())
			wg.Done()
			return
		default:
			if len(statsChans.chans) >= numReportedPids {
				wg.Done()
				return
			}
			dps, err := fn(conf)
			if err != nil {
				logger.Error(err)
				wg.Done()
				return
			}
			if len(dps) > 0 {
				select {
				case statsChans.getChan(dps[0].Dimensions["process_num"]) <- dps:
					logger.Debugf("succeeded to write 'show stats' datapoints to channel for process number %s", dps[0].Dimensions["process_num"])
				default:
				}
			}
		}
	}
}

// Creates datapoints from csv stats fetched through http.
func (m *Monitor) fetchHTTP(conf *Config) ([]*datapoint.Datapoint, error) {
	body, err := httpReader(conf)
	defer closeBody(body)
	if err != nil {
		return nil, fmt.Errorf("cannot scrape HAProxy stats: %+v", err)
	}
	return m.fetchCsv(body), nil
}

func httpReader(conf *Config) (io.ReadCloser, error) {
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

func closeBody(body io.ReadCloser) {
	if body != nil {
		body.Close()
	}
}

// Creates datapoints from csv stats reader.
func (m *Monitor) fetchCsv(body io.Reader) []*datapoint.Datapoint {
	dps := make([]*datapoint.Datapoint, 0)
	for _, metricValuePairs := range readCsv(body) {
		if len(m.proxies) != 0 && !m.proxies[metricValuePairs["pxname"]] && !m.proxies[metricValuePairs["svname"]] {
			continue
		}
		for metric, value := range metricValuePairs {
			if dp := toDp(sfxMetricsMap[metric], value); dp != nil {
				dp.Dimensions["proxy_name"] = metricValuePairs["pxname"]
				dp.Dimensions["service_name"] = metricValuePairs["svname"]
				// WARNING: Both pid and Process_num are indexes identifying HAProxy processes. pid in the context of
				// proxy stats and Process_num in the context of HAProxy process info. It says in the docs
				// https://cbonte.github.io/haproxy-dconv/1.8/management.html#9.1 that pid is zero-based. But, it is
				// exactly the same as Process_num, a natural number. We therefore assign pid and Process_num to
				// dimension process_num without modifying them to match.
				dp.Dimensions["process_num"] = metricValuePairs["pid"]
				dps = append(dps, dp)
			}
		}
	}
	return dps
}

func readCsv(body io.Reader) map[int]map[string]string {
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

func toDp(metric string, value string) *datapoint.Datapoint {
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

// Creates datapoints for all HAProxy processes from stats and info fetched through socket.
func (m *Monitor) fetchAllSocket(ctx context.Context, conf *Config, numReportedPids int) []*datapoint.Datapoint {
	var statsDpsChans, infoDpsChans dpsChans
	var wg sync.WaitGroup
	for i := 0; i < min1(maxRoutines, numReportedPids); i++ {
		wg.Add(1)
		go fetchRepeatStats(ctx, m.fetchSocket, conf, numReportedPids, &wg, &statsDpsChans)
		wg.Add(1)
		go m.fetchRepeatInfo(ctx, conf, numReportedPids, &wg, &infoDpsChans)
	}
	wg.Wait()
	dps := make([]*datapoint.Datapoint, 0)
	for _, v := range statsDpsChans.chans {
		dps = append(dps, <-v...)
	}
	for _, v := range infoDpsChans.chans {
		dps = append(dps, <-v...)
	}
	return dps
}

// Creates datapoints from csv stats fetched from unix socket.
func (m *Monitor) fetchSocket(conf *Config) ([]*datapoint.Datapoint, error) {
	body, err := socketReader(m.url, "show stat\n", conf.Timeout)
	defer closeBody(body)
	if err != nil {
		return nil, fmt.Errorf("cannot scrape HAProxy stats: %+v", err)
	}
	return m.fetchCsv(body), nil
}

// Writes into channel, info datapoints fetch through unix socket command 'show info'.
func (m *Monitor) fetchRepeatInfo(ctx context.Context, conf *Config, numReportedPids int, wg *sync.WaitGroup, infoChans *dpsChans) {
	timer := time.NewTicker(repeatFetchAfter)
	defer timer.Stop()
	for range timer.C {
		select {
		case <-ctx.Done():
			logger.Errorf("failed to write 'show info' datapoints to channel: %+v", ctx.Err())
			wg.Done()
			return
		default:
			if len(infoChans.chans) >= numReportedPids {
				wg.Done()
				return
			}
			dps := make([]*datapoint.Datapoint, 0)
			metricValuePairs, err := m.readInfoOutput(conf)
			if err != nil {
				logger.Error(err)
				wg.Done()
				return
			}
			for metric, value := range metricValuePairs {
				if dp := toDp(sfxMetricsMap[metric], value); dp != nil {
					// WARNING: Both pid and Process_num are indexes identifying HAProxy processes. pid in the context of
					// proxy stats and Process_num in the context of HAProxy process info. It says in the docs
					// https://cbonte.github.io/haproxy-dconv/1.8/management.html#9.1 that pid is zero-based. But, it is
					// exactly the same as Process_num, a natural number. We therefore assign pid and Process_num to
					// dimension process_num without modifying them to match.
					dp.Dimensions["process_num"] = metricValuePairs["Process_num"]
					dps = append(dps, dp)
				}
			}
			if len(dps) > 0 {
				select {
				case infoChans.getChan(dps[0].Dimensions["process_num"]) <- dps:
					logger.Debugf("succeeded to write stats datapoints to channel for process number %s", dps[0].Dimensions["process_num"])
				default:
				}
			}
		}
	}
}

// Creates a map of the 'show info' unix socket command output.
func (m *Monitor) readInfoOutput(conf *Config) (map[string]string, error) {
	body, err := socketReader(m.url, "show info\n", conf.Timeout)
	defer closeBody(body)
	if err != nil {
		return nil, fmt.Errorf("cannot scrape HAProxy stats: %+v", err)
	}
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
	return row, nil
}

func socketReader(u *url.URL, cmd string, timeout time.Duration) (io.ReadCloser, error) {
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

// Returns the min parameter with 1 as the absolute min
func min1(a, b int) int {
	if a < b && a > 0 {
		return a
	} else if b > 0 {
		return b
	}
	return 1
}
