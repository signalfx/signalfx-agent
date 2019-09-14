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

	"github.com/signalfx/signalfx-agent/internal/utils"

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
)

type chans struct {
	dpChans map[string]chan []*datapoint.Datapoint
	lock    sync.Mutex
}

// Merges proxy stats datapoints from workers fetched through http.
func mergeHTTP(ctx context.Context, conf *Config, numProcesses int) []*datapoint.Datapoint {
	var proxyStatsChans chans
	var wg sync.WaitGroup
	for i := 0; i < numProcesses; i++ {
		wg.Add(1)
		go proxyStatsWorker(ctx, fetchProxyStatsHTTP, conf, &wg, &proxyStatsChans)
	}
	wg.Wait()
	dps := make([]*datapoint.Datapoint, 0)
	for _, v := range proxyStatsChans.dpChans {
		dps = append(dps, <-v...)
		close(v)
	}
	return dps
}

// Merges proxy stats datapoints from workers fetched from unix socket.
func mergeSocket(ctx context.Context, conf *Config, numProcesses int) []*datapoint.Datapoint {
	var proxyStatsChans, processInfoChans chans
	var wg sync.WaitGroup
	for i := 0; i < numProcesses; i++ {
		wg.Add(1)
		go proxyStatsWorker(ctx, fetchProxyStatsSocket, conf, &wg, &proxyStatsChans)
		wg.Add(1)
		go processInfoWorker(ctx, conf, &wg, &processInfoChans)
	}
	wg.Wait()
	dps := make([]*datapoint.Datapoint, 0)
	for _, v := range proxyStatsChans.dpChans {
		dps = append(dps, <-v...)
		close(v)
	}
	for _, v := range processInfoChans.dpChans {
		dps = append(dps, <-v...)
		close(v)
	}
	return dps
}

// Writes fetched proxy stats datapoints of a process into channel.
func proxyStatsWorker(ctx context.Context, fn func(conf *Config) ([]*datapoint.Datapoint, error), conf *Config, wg *sync.WaitGroup, statsChans *chans) {
	timer := time.NewTicker(repeatFetchAfter)
	defer timer.Stop()
	for range timer.C {
		select {
		case <-ctx.Done():
			logger.Errorf("could not write 'show stats' datapoints to channel: %+v", ctx.Err())
			wg.Done()
			return
		default:
			dps, err := fn(conf)
			if err != nil {
				logger.Error(err)
				wg.Done()
				return
			}
			select {
			case statsChans.getChan(dps[0].Dimensions["process_num"]) <- dps:
				logger.Debugf("wrote 'show stats' datapoints to channel for process number %s", dps[0].Dimensions["process_num"])
				wg.Done()
				return
			default:
			}
		}
	}
}

// Writes fetched 'show info' datapoints of a process into channel.
func processInfoWorker(ctx context.Context, conf *Config, wg *sync.WaitGroup, infoChans *chans) {
	timer := time.NewTicker(repeatFetchAfter)
	defer timer.Stop()
	for range timer.C {
		select {
		case <-ctx.Done():
			logger.Errorf("could not write 'show info' datapoints to channel. %+v", ctx.Err())
			wg.Done()
			return
		default:
			dps := make([]*datapoint.Datapoint, 0)
			infoPairs, err := readProcessInfoOutput(conf)
			if err != nil {
				logger.Error(err)
				wg.Done()
				return
			}
			for metric, value := range infoPairs {
				if dp := newDp(sfxMetricsMap[metric], value); dp != nil {
					// WARNING: Both pid and Process_num are indexes identifying HAProxy processes. pid in the context of
					// proxy stats and Process_num in the context of HAProxy process info. It says in the docs
					// https://cbonte.github.io/haproxy-dconv/1.8/management.html#9.1 that pid is zero-based. But, it is
					// exactly the same as Process_num, a natural number. We therefore assign pid and Process_num to
					// dimension process_num without modifying them to match.
					dp.Dimensions["process_num"] = infoPairs["Process_num"]
					dps = append(dps, dp)
				}
			}
			select {
			case infoChans.getChan(dps[0].Dimensions["process_num"]) <- dps:
				logger.Debugf("wrote stats datapoints to channel for process number %s", dps[0].Dimensions["process_num"])
				wg.Done()
				return
			default:
			}
		}
	}
}

// Fetches proxy stats datapoints through http.
func fetchProxyStatsHTTP(conf *Config) ([]*datapoint.Datapoint, error) {
	return fetchProxyStats(conf, httpReader, "GET")
}

// Fetches proxy stats datapoints from unix socket.
func fetchProxyStatsSocket(conf *Config) ([]*datapoint.Datapoint, error) {
	return fetchProxyStats(conf, socketReader, "show stat\n")
}

// Creates datapoints from fetched proxy stats csv map.
func fetchProxyStats(conf *Config, reader func(*Config, string) (io.ReadCloser, error), cmd string) ([]*datapoint.Datapoint, error) {
	body, err := reader(conf, cmd)
	defer closeBody(body)
	if err != nil {
		return nil, fmt.Errorf("could not scrape HAProxy stats: %+v", err)
	}
	dps := make([]*datapoint.Datapoint, 0)
	csvMap, err := readProxyStats(body)
	if err != nil {
		return nil, err
	}
	for _, headerValuePairs := range csvMap {
		if len(conf.Proxies) != 0 && !conf.hasProxies(headerValuePairs["pxname"], headerValuePairs["svname"]) {
			continue
		}
		for metric, value := range headerValuePairs {
			if dp := newDp(sfxMetricsMap[metric], value); dp != nil {
				dp.Dimensions["proxy_name"] = headerValuePairs["pxname"]
				dp.Dimensions["service_name"] = headerValuePairs["svname"]
				// WARNING: Both pid and Process_num are indexes identifying HAProxy processes. pid in the context of
				// proxy stats and Process_num in the context of HAProxy process info. It says in the docs
				// https://cbonte.github.io/haproxy-dconv/1.8/management.html#9.1 that pid is zero-based. But, it is
				// exactly the same as Process_num, a natural number. We therefore assign pid and Process_num to
				// dimension process_num without modifying them to match.
				dp.Dimensions["process_num"] = headerValuePairs["pid"]
				dps = append(dps, dp)
			}
		}
	}
	if len(dps) == 0 {
		return nil, fmt.Errorf("zero stats datapoints returned")
	}
	return dps, nil
}

// Reads proxy stats in csv format and converts the csv data to map.
func readProxyStats(body io.Reader) (map[int]map[string]string, error) {
	r := csv.NewReader(body)
	r.TrimLeadingSpace = true
	r.TrailingComma = true
	rows := map[int]map[string]string{}
	table, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(table) < 2 || utils.TrimAllSpaces(table[0][0]) != "#pxname" {
		return nil, fmt.Errorf("incompatible csv data returned")
	}
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
	return rows, nil
}

// Reads 'show info' command output and converts the output to map.
func readProcessInfoOutput(conf *Config) (map[string]string, error) {
	body, err := socketReader(conf, "show info\n")
	defer closeBody(body)
	if err != nil {
		return nil, fmt.Errorf("could not scrape HAProxy stats: %+v", err)
	}
	sc := bufio.NewScanner(body)
	processInfoOutput := map[string]string{}
	for sc.Scan() {
		s := strings.SplitN(sc.Text(), ":", 2)
		if len(s) != 2 || strings.TrimSpace(s[0]) == "" || strings.TrimSpace(s[1]) == "" {
			logger.Debugf("did not get exactly 2 substrings after splitting string '%s' using separator ':'", sc.Text())
			continue
		}
		processInfoOutput[strings.TrimSpace(s[0])] = strings.TrimSpace(s[1])
	}
	if len(processInfoOutput) == 0 {
		return nil, fmt.Errorf("zero process info datapoints returned")
	}
	return processInfoOutput, nil
}

// Creates datapoints from proxy stats and 'show info' command output string values.
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

func httpReader(conf *Config, method string) (io.ReadCloser, error) {
	client := http.Client{
		Timeout:   conf.Timeout,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: !conf.SSLVerify}},
	}
	req, err := http.NewRequest(method, conf.URL, nil)
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

func socketReader(conf *Config, cmd string) (io.ReadCloser, error) {
	u, err := url.Parse(conf.URL)
	if err != nil {
		return nil, fmt.Errorf("cannot parse url %s status. %v", conf.URL, err)
	}
	f, err := net.DialTimeout("unix", u.Path, conf.Timeout)
	if err != nil {
		return nil, err
	}
	if err := f.SetDeadline(time.Now().Add(conf.Timeout)); err != nil {
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

func closeBody(body io.ReadCloser) {
	if body != nil {
		body.Close()
	}
}

func (d *chans) getChan(k string) chan []*datapoint.Datapoint {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.dpChans == nil {
		d.dpChans = make(map[string]chan []*datapoint.Datapoint)
	}
	if d.dpChans[k] == nil {
		d.dpChans[k] = make(chan []*datapoint.Datapoint, 1)
	}
	return d.dpChans[k]
}
