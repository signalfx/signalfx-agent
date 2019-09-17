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
	"ConnRate":           haproxyConnectionRateAll,
}

const (
	tickInterval = 1000 * time.Millisecond
)

var jobsLock sync.Mutex

// Merges proxy stats datapoints of processes fetched through http.
func fetchHTTP(ctx context.Context, conf *Config, numProcesses int, proxies map[string]bool) []*datapoint.Datapoint {
	jobs := make(map[string]bool, numProcesses)
	dpsChans := make([]chan []*datapoint.Datapoint, numProcesses)
	var wg sync.WaitGroup
	for i := 0; i < numProcesses; i++ {
		i := i
		dpsChans[i] = make(chan []*datapoint.Datapoint, 1)
		wg.Add(1)
		go statsJob(ctx, httpJob, conf, &wg, dpsChans[i], jobs, proxies)
	}
	wg.Wait()
	dps := make([]*datapoint.Datapoint, 0)
	for _, v := range dpsChans {
		dps = append(dps, <-v...)
	}
	return dps
}

// Merges proxy stats datapoints of processes fetched from unix socket.
func fetchSocket(ctx context.Context, conf *Config, numProcesses int, proxies map[string]bool) []*datapoint.Datapoint {
	jobs := make(map[string]bool, 2*numProcesses)
	dpsChans := make([]chan []*datapoint.Datapoint, 2*numProcesses)
	var wg sync.WaitGroup
	for i := 0; i < numProcesses; i++ {
		i := i
		dpsChans[i] = make(chan []*datapoint.Datapoint, 1)
		dpsChans[i+numProcesses] = make(chan []*datapoint.Datapoint, 1)
		wg.Add(2)
		go statsJob(ctx, socketJob, conf, &wg, dpsChans[i], jobs, proxies)
		go infoJob(ctx, conf, &wg, dpsChans[i+numProcesses], jobs)
	}
	wg.Wait()
	dps := make([]*datapoint.Datapoint, 0)
	for _, v := range dpsChans {
		dps = append(dps, <-v...)
	}
	return dps
}

// Writes http and socket fetched proxy stats datapoints of a process into dps channel.
func statsJob(ctx context.Context, fn func(*Config, map[string]bool) ([]*datapoint.Datapoint, error), conf *Config, wg *sync.WaitGroup, dpsChan chan []*datapoint.Datapoint, jobs map[string]bool, proxies map[string]bool) {
	defer wg.Done()
	defer close(dpsChan)
	timer := time.NewTicker(tickInterval)
	defer timer.Stop()
	for range timer.C {
		select {
		case <-ctx.Done():
			logger.Debugf("Failed to write proxy stats datapoints to channel: %+v", ctx.Err())
			return
		default:
			dps, err := fn(conf, proxies)
			if err != nil {
				logger.Errorf("Failed to scrape proxy stats: %+v", err)
				return
			}
			jobKey := "stats" + dps[0].Dimensions["process_num"]
			if updateChan(jobs, jobKey, dpsChan, dps) {
				return
			}
		}
	}
}

// Writes socket fetched process info datapoints of a process into dps channel.
func infoJob(ctx context.Context, conf *Config, wg *sync.WaitGroup, dpsChan chan []*datapoint.Datapoint, jobs map[string]bool) {
	defer wg.Done()
	defer close(dpsChan)
	timer := time.NewTicker(tickInterval)
	defer timer.Stop()
	for range timer.C {
		select {
		case <-ctx.Done():
			logger.Debugf("Failed to write process info datapoints to channel. %+v", ctx.Err())
			return
		default:
			dps := make([]*datapoint.Datapoint, 0)
			infoPairs, err := infoMap(conf)
			if err != nil {
				logger.Errorf("Failed to scrape process info data: %+v", err)
				return
			}
			for metric, value := range infoPairs {
				if dp := newDp(sfxMetricsMap[metric], value); dp != nil {
					// WARNING: Both pid and Process_num are HAProxy process identifiers. pid in the context of
					// proxy stats and Process_num in the context of HAProxy process info. It says in the docs
					// https://cbonte.github.io/haproxy-dconv/1.8/management.html#9.1 that pid is zero-based. But, we
					// find that pid is exactly the same as Process_num, a natural number. We therefore assign pid and
					// Process_num to dimension process_num without modifying them to match.
					dp.Dimensions["process_num"] = infoPairs["Process_num"]
					dps = append(dps, dp)
				}
			}
			jobKey := "info" + dps[0].Dimensions["process_num"]
			if updateChan(jobs, jobKey, dpsChan, dps) {
				return
			}
		}
	}
}

// Fetches proxy stats datapoints of a process through http.
func httpJob(conf *Config, proxies map[string]bool) ([]*datapoint.Datapoint, error) {
	return jobHelper(conf, httpReader, "GET", proxies)
}

// Fetches proxy stats datapoints of a process from unix socket.
func socketJob(conf *Config, proxies map[string]bool) ([]*datapoint.Datapoint, error) {
	return jobHelper(conf, socketReader, "show stat\n", proxies)
}

// A second order function for taking http and socket functions that fetch proxy stats datapoints of a process.
func jobHelper(conf *Config, reader func(*Config, string) (io.ReadCloser, error), cmd string, proxies map[string]bool) ([]*datapoint.Datapoint, error) {
	body, err := reader(conf, cmd)
	defer closeBody(body)
	if err != nil {
		return nil, err
	}
	dps := make([]*datapoint.Datapoint, 0)
	csvMap, err := statsMap(body)
	if err != nil {
		return nil, err
	}
	for _, headerValuePairs := range csvMap {
		if len(proxies) != 0 && !proxies[headerValuePairs["pxname"]] && !proxies[headerValuePairs["svname"]] {
			continue
		}
		for metric, value := range headerValuePairs {
			if dp := newDp(sfxMetricsMap[metric], value); dp != nil {
				dp.Dimensions["proxy_name"] = headerValuePairs["pxname"]
				dp.Dimensions["service_name"] = headerValuePairs["svname"]
				// WARNING: Both pid and Process_num are HAProxy process identifiers. pid in the context of
				// proxy stats and Process_num in the context of HAProxy process info. It says in the docs
				// https://cbonte.github.io/haproxy-dconv/1.8/management.html#9.1 that pid is zero-based. But, we
				// find that pid is exactly the same as Process_num, a natural number. We therefore assign pid and
				// Process_num to dimension process_num without modifying them to match.
				dp.Dimensions["process_num"] = headerValuePairs["pid"]
				dps = append(dps, dp)
			}
		}
	}
	if len(dps) == 0 {
		return nil, fmt.Errorf("failed to create proxy stats datapoints")
	}
	return dps, nil
}

// Fetches and convert proxy stats in csv format to map.
func statsMap(body io.Reader) (map[int]map[string]string, error) {
	r := csv.NewReader(body)
	r.TrimLeadingSpace = true
	r.TrailingComma = true
	rows := map[int]map[string]string{}
	table, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(table) == 0 {
		return nil, fmt.Errorf("unavailable proxy stats csv data")
	}
	if utils.TrimAllSpaces(table[0][0]) != "#pxname" {
		return nil, fmt.Errorf("incompatible proxy stats csv data. Expected '#pxname' as first header")
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

// Fetches and converts process info (i.e. 'show info' command output) to map.
func infoMap(conf *Config) (map[string]string, error) {
	body, err := socketReader(conf, "show info\n")
	defer closeBody(body)
	if err != nil {
		return nil, err
	}
	sc := bufio.NewScanner(body)
	processInfoOutput := map[string]string{}
	for sc.Scan() {
		s := strings.SplitN(sc.Text(), ":", 2)
		if len(s) != 2 || strings.TrimSpace(s[0]) == "" || strings.TrimSpace(s[1]) == "" {
			continue
		}
		processInfoOutput[strings.TrimSpace(s[0])] = strings.TrimSpace(s[1])
	}
	if len(processInfoOutput) == 0 {
		return nil, fmt.Errorf("failed to create process info datapoints")
	}
	return processInfoOutput, nil
}

// Creates datapoint from proxy stats and process info key value pairs.
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

func updateChan(jobs map[string]bool, jobKey string, dpsChan chan []*datapoint.Datapoint, dps []*datapoint.Datapoint) bool {
	jobsLock.Lock()
	defer jobsLock.Unlock()
	if !jobs[jobKey] {
		dpsChan <- dps
		jobs[jobKey] = true
		return true
	}
	return false
}
