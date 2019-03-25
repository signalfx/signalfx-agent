package expvar

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	url2 "net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const monitorType = "expvar"

// MONITOR(expvar): This monitor exports metrics derived from [expvar](https://golang.org/pkg/expvar/)
// variables in JSON objects retrieved from a HTTP endpoint.
//
// The monitor uses configured JSON paths to get metric and dimension values from the JSON objects.
// [Memstats metrics](https://github.com/signalfx/integrations/blob/master/expvar/docs/expvar_metrics.md)
// are created by default. They do not require configuration. They are derived from variable
// [memstats](https://golang.org/pkg/runtime/) that is exposed by default.
//
// Below is a sample YAML configuration showing the minimal configuration of the expvar monitor for
// exporting memstats metrics. The `extraDimensions` field is not required but recommended.
//```
// monitors:
// - type: expvar
//   host: 172.17.0.3
//   path: /debug/vars
//   port: 8000
//   extraDimensions:
//     metric_source: expvar
//```
//
// Below is a sample YAML configuration showing the configuration of custom metrics `xyz` and `h.i`.
// `name` is an optional configuration for metrics and dimensions. If not provided then the required
// `JSONPath` is used for `name`.
//```
// monitors:
// - type: expvar
//   host: 172.12.0.6
//   path: /debug/vars
//   port: 8123
//     metrics:
//       - name: xyz
//         JSONPath: a.b.c.d
//         type: counter
//         dimensions:
//           - name: rst
//             JSONPath: a.b
//       - JSONPath: h.i
//         type: gauge
//       - JSONPath: j
//         type: gauge
//         dimensions:
//           - name: kl
//             value: mno
//   extraDimensions:
//     metric_source: expvar
//```
//
// Below is an example JSON object that the above configuration may apply to.
//```
//{
//  "a": {
//    "b": {
//      "c": [
//        {
//          "d": 4
//        },
//        {
//          "d": 9
//        },
//      ]
//    }
//  },
//  "h": [
//    {
//      "i": 1.2
//    }
//  ],
//  "j" : 0.8
//}
//```
// Two data points will be created for the metric `a.b.c.d`. The first data point will have the value 4 and
// dimensions `a.b: c`, `a.b.c: 0` and `metric_source: expvar`. The second data point will have the value 9
// and dimensions `a.b: c`, `a.b.c: 1` and `metric_source: expvar`. The monitor creates dimensions such as
// `a.b.c` to store array indexes.
//
// For metric `h.i`, a data point of value 1.2, dimensions `h: 0` and `metric_source: expvar` will be created.
//
// For metric `j`, a data point of value 0.8, dimensions `kl: mno` and `metric_source: expvar` will be created.
// Note that the value for dimension `kl` was provided.
//
// The path of a dimension must follow the path of its metric and be shorter.

const (
	gauge                                  = "gauge"
	counter                                = "counter"
	memstatsPauseNsMetricPath              = "memstats.PauseNs"
	memstatsPauseEndMetricPath             = "memstats.PauseEnd"
	memstatsNumGCMetricPath                = "memstats.NumGC"
	memstatsMostRecentGCPauseNsMetricName  = "memstats.most_recent_gc_pause_ns"
	memstatsMostRecentGCPauseEndMetricName = "memstats.most_recent_gc_pause_end"
	memstatsBySizeSizeMetricPath           = "memstats.BySize.Size"
	memstatsBySizeMallocsMetricPath        = "memstats.BySize.Mallocs"
	memstatsBySizeFreesMetricPath          = "memstats.BySize.Frees"
	memstatsBySizeDimensionPath            = "memstats.BySize"
)

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

//
type metric struct {
	// Metric name
	Name string `yaml:"name"`
	// JSON path of metric value
	JSONPath string `yaml:"JSONPath" validate:"required"`
	// SignalFx metric type. Possible values are "gauge" or "counter"
	Type  string `yaml:"type" validate:"required"`
	Dims  []*dim `yaml:"dimensions"`
	_type datapoint.MetricType
	// Slice of Path substrings separated by .
	keys []string
}

func (met *metric) dims() map[string]string {
	var dims map[string]string
	if len(met.Dims) > 0 {
		dims = make(map[string]string, len(met.Dims))
		for _, _dim := range met.Dims {
			if strings.TrimSpace(_dim.Name) != "" && strings.TrimSpace(_dim.Value) != "" {
				dims[_dim.Name] = _dim.Value
			}
		}
	}
	return dims
}

type dim struct {
	// Dimension name
	Name string `yaml:"name"`
	// JSON path of dimension value
	JSONPath string `yaml:"JSONPath"`
	// Dimension value
	Value string `yaml:"value"`
	// Slice of Path substrings separated by .
	splits []string
}

// Exclude EnableGC, DebugGC,

// Config for this monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	// Host of the exporter
	Host string `yaml:"host" validate:"required"`
	// Port of the exporter
	Port uint16 `yaml:"port" validate:"required"`
	// If true, the agent will connect to the exporter using HTTPS instead of plain HTTP.
	UseHTTPS bool `yaml:"useHTTPS"`
	// If useHTTPS is true and this option is also true, the exporter's TLS cert will not be verified.
	SkipVerify bool `yaml:"skipVerify"`
	// Path to the metrics endpoint on the exporter server, usually `/debug/vars` (the default).
	Path            string `yaml:"path" default:"/debug/vars"`
	EnhancedMetrics bool   `yaml:"enhancedMetrics"`
	// Custom metrics
	Metrics []*metric `yaml:"metrics"`
	// Default metrics
	url         *url2.URL
	runInterval time.Duration
}

// Monitor for Expvar metrics
type Monitor struct {
	Output types.Output
	cancel func()
	ctx    context.Context
	client *http.Client
}

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Validate validates configuration
func (conf *Config) Validate() error {
	if conf.Metrics != nil {
		for _, m := range conf.Metrics {
			_type := strings.TrimSpace(strings.ToLower(m.Type))
			if _type != gauge && _type != counter {
				return fmt.Errorf("unsupported metric type %s. Supported metric types are: %s, %s", m.Type, gauge, counter)
			}
			for _, d := range m.Dims {
				if d != nil && strings.TrimSpace(d.JSONPath) != "" && !strings.HasPrefix(m.JSONPath, d.JSONPath) {
					return fmt.Errorf("invalid dimension path %s. Dimension path not parent path of metric path %s", d.JSONPath, m.JSONPath)
				}
			}
		}
	}
	return nil
}

// Configure monitor
func (m *Monitor) Configure(conf *Config) error {
	conf.setURL()
	conf.setRunInterval()
	conf.initMetrics()
	m.setContext()
	m.setClient()
	utils.RunOnInterval(m.ctx, func() {
		dpsMap, err := m.fetchMetrics(conf)
		if err != nil {
			logger.WithError(err).Error("could not get expvar metrics")
			return
		}
		mostRecentGCPauseIndex := getMostRecentGCPauseIndex(dpsMap)
		now := time.Now()
		for metricPath, dps := range *dpsMap {
			for _, dp := range dps {
				if err := m.sendDatapoint(conf, dp, metricPath, mostRecentGCPauseIndex, &now); err != nil {
					logger.Error(err)
				}
			}
		}
	}, conf.runInterval)
	return nil
}

func (m *Monitor) sendDatapoint(conf *Config, dp *datapoint.Datapoint, metricPath string, mostRecentGCPauseIndex int64, now *time.Time) (err error) {
	if metricPath == memstatsPauseNsMetricPath || metricPath == memstatsPauseEndMetricPath {
		var index int64
		if index, err = strconv.ParseInt(dp.Dimensions[metricPath], 10, 0); err == nil && index == mostRecentGCPauseIndex {
			dp.Metric = memstatsMostRecentGCPauseNsMetricName
			if metricPath == memstatsPauseEndMetricPath {
				dp.Metric = memstatsMostRecentGCPauseEndMetricName
			}
			// For metrics memstats.PauseNs and memstats.PauseEnd the key of dimensions for is equal metric path
			delete(dp.Dimensions, metricPath)
		} else {
			if err != nil {
				err = fmt.Errorf("failed to set metric most recent GC pause metric. %+v", err)
			}
			return err
		}
	}
	// Renaming dimension to class
	if metricPath == memstatsBySizeSizeMetricPath || metricPath == memstatsBySizeMallocsMetricPath || metricPath == memstatsBySizeFreesMetricPath {
		dp.Dimensions["class"] = dp.Dimensions[memstatsBySizeDimensionPath]
		delete(dp.Dimensions, memstatsBySizeDimensionPath)
	}
	dp.Timestamp = *now
	dp.Dimensions["process_host"] = conf.Host
	dp.Dimensions["process_expvar_port"] = strconv.FormatInt(int64(conf.Port), 10)
	dp.Dimensions["plugin"] = monitorType
	m.Output.SendDatapoint(dp)
	return nil
}

func (conf *Config) setURL() {
	conf.url = &url2.URL{
		Scheme: func() string {
			if conf.UseHTTPS {
				return "https"
			}
			return "http"
		}(),
		Host: conf.Host + ":" + fmt.Sprint(conf.Port),
		Path: conf.Path,
	}
}

func (conf *Config) setRunInterval() {
	conf.runInterval = time.Duration(conf.IntervalSeconds) * time.Second
}

func (conf *Config) initMetrics() {
	// Default metrics paths
	memstatsMetricPathsGauge := []string{"memstats.HeapAlloc", "memstats.HeapSys", "memstats.HeapIdle", "memstats.HeapInuse", "memstats.HeapReleased", "memstats.HeapObjects", "memstats.StackInuse", "memstats.StackSys", "memstats.MSpanInuse", "memstats.MSpanSys", "memstats.MCacheInuse", "memstats.MCacheSys", "memstats.BuckHashSys", "memstats.GCSys", "memstats.OtherSys", "memstats.Sys", "memstats.NextGC", "memstats.LastGC", "memstats.GCCPUFraction", "memstats.EnableGC"}
	memstatsMetricPathsCounter := []string{"memstats.TotalAlloc", "memstats.Lookups", "memstats.Mallocs", "memstats.Frees", "memstats.PauseTotalNs", memstatsNumGCMetricPath, "memstats.NumForcedGC"}

	if conf.EnhancedMetrics {
		memstatsMetricPathsGauge = append(memstatsMetricPathsGauge, "memstats.Alloc", memstatsPauseNsMetricPath, memstatsPauseEndMetricPath)
		memstatsMetricPathsCounter = append(memstatsMetricPathsCounter, memstatsBySizeSizeMetricPath, memstatsBySizeMallocsMetricPath, memstatsBySizeFreesMetricPath)
	}
	if conf.Metrics == nil {
		conf.Metrics = make([]*metric, 0, len(memstatsMetricPathsGauge)+len(memstatsMetricPathsCounter))
	}
	// Initializing custom metrics
	for _, m := range conf.Metrics {
		if strings.TrimSpace(m.Name) == "" {
			m.Name = toSnakeCase(m.JSONPath)
		}
		m._type = func() datapoint.MetricType {
			if m.Type == gauge {
				return datapoint.Gauge
			}
			return datapoint.Counter
		}()
		if m.Dims == nil {
			m.Dims = []*dim{{}}
		}
		m.keys = strings.Split(m.JSONPath, ".")
	}
	// Initializing default metrics
	for _, path := range memstatsMetricPathsGauge {
		conf.Metrics = append(conf.Metrics, &metric{Name: toSnakeCase(path), JSONPath: path, Type: gauge, _type: datapoint.Gauge, Dims: []*dim{{}}, keys: strings.Split(path, ".")})
	}
	for _, path := range memstatsMetricPathsCounter {
		conf.Metrics = append(conf.Metrics, &metric{Name: toSnakeCase(path), JSONPath: path, Type: counter, _type: datapoint.Counter, Dims: []*dim{{}}, keys: strings.Split(path, ".")})
	}
}

func (m *Monitor) setContext() {
	m.ctx, m.cancel = context.WithCancel(context.Background())
}

func (m *Monitor) setClient() {
	m.client = &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
		},
	}
}

func (m *Monitor) fetchMetrics(conf *Config) (*map[string][]*datapoint.Datapoint, error) {
	resp, err := m.client.Get(conf.url.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	metricsJSON := interface{}(nil)
	err = json.Unmarshal(body, &metricsJSON)
	if err != nil {
		return nil, err
	}
	var dpsMap map[string][]*datapoint.Datapoint
	var processName string
	if _map, ok := metricsJSON.(map[string]interface{}); ok {
		if processName, err = getProcessName(_map); err != nil {
			logger.Error(err)
		}
		dpsMap = make(map[string][]*datapoint.Datapoint)
		for _, _metric := range conf.Metrics {
			dp := datapoint.Datapoint{Dimensions: map[string]string{}}
			if processName != "" {
				dp.Dimensions["process_name"] = processName
			}
			dpsMap[_metric.JSONPath] = make([]*datapoint.Datapoint, 0)
			setDatapoints(_map[_metric.keys[0]], _metric, &dp, &dpsMap, 0)
		}
	}
	return &dpsMap, nil
}

// getMostRecentGCPauseIndex logic is derived from https://golang.org/pkg/runtime/ in the PauseNs section of the 'type MemStats' section
func getMostRecentGCPauseIndex(dpsMap *map[string][]*datapoint.Datapoint) int64 {
	dps := (*dpsMap)[memstatsNumGCMetricPath]
	mostRecentGCPauseIndex := int64(-1)
	if len(dps) > 0 && dps[0].Value != nil {
		if numGC, err := strconv.ParseInt(dps[0].Value.String(), 10, 0); err == nil {
			mostRecentGCPauseIndex = (numGC + 255) % 256
		}
	}
	return mostRecentGCPauseIndex
}

var slashLastRegexp = regexp.MustCompile("[^\\/]*$")

func getProcessName(_map map[string]interface{}) (string, error) {
	if cmdline, ok := _map["cmdline"].([]interface{}); ok && len(cmdline) > 0 {
		if processName := strings.TrimSpace(slashLastRegexp.FindStringSubmatch(cmdline[0].(string))[0]); processName != "" {
			return processName, nil
		}
	}
	return "", fmt.Errorf("failed to get process name from the first array value of cmdline in map: %+v", _map)
}

// setDatapoints follows metric path down the v map setting dimensions along the way and metric value in the end
func setDatapoints(v interface{}, _metric *metric, dp *datapoint.Datapoint, dpsMap *map[string][]*datapoint.Datapoint, i int) {
	if _map, ok := v.(map[string]interface{}); ok {
		for _, _dim := range _metric.Dims {
			if len(_dim.splits) != 0 && len(_dim.splits) == i {
				dp.Dimensions[_dim.Name] = _metric.keys[i]
			}
		}
		setDatapoints(_map[_metric.keys[i+1]], _metric, dp, dpsMap, i+1)
	} else if arr, ok := v.([]interface{}); ok {
		_dp := dp
		for _i, _v := range arr {
			if _i > 0 {
				_dp = &datapoint.Datapoint{Dimensions: copyDims(dp)}
			}
			newDim := true
			for _, _dim := range _metric.Dims {
				if len(_dim.splits) == i+1 {
					_dp.Dimensions[_dim.Name] = strconv.Itoa(_i)
					newDim = false
				}
			}
			if newDim {
				_dp.Dimensions[strings.Join(_metric.keys[:(i+1)], ".")] = strconv.Itoa(_i)
			}
			setDatapoints(_v, _metric, _dp, dpsMap, i)
		}
	} else {
		if v == nil {
			logger.Errorf("failed to find value for metric %s with JSON path %s", _metric.Name, _metric.JSONPath)
			return
		}
		dp.Metric, dp.MetricType = _metric.Name, _metric._type
		for _, _dim := range _metric.Dims {
			if strings.TrimSpace(_dim.Name) != "" && strings.TrimSpace(_dim.Value) != "" {
				dp.Dimensions[_dim.Name] = _dim.Value
			}
		}
		var err error
		if dp.Value, err = datapoint.CastMetricValue(v); err == nil {
			(*dpsMap)[_metric.JSONPath] = append((*dpsMap)[_metric.JSONPath], dp)
		} else {
			logger.Errorf("failed to set value for metric %s with JSON path %s because of type conversion error due to %+v", _metric.Name, _metric.JSONPath, err)
			return
		}
	}
}

func copyDims(dp *datapoint.Datapoint) map[string]string {
	dims := map[string]string{}
	for k, v := range dp.Dimensions {
		dims[k] = v
	}
	return dims
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

var camelRegexp = regexp.MustCompile("(^[^A-Z]*|[A-Z]*)([A-Z][^A-Z]+|$)")

func toSnakeCase(s string) string {
	snake := ""
	for _, split := range strings.Split(s, ".") {
		for _, submatches := range camelRegexp.FindAllStringSubmatch(split, -1) {
			for _, submatch := range submatches[1:] {
				submatch = strings.TrimSpace(submatch)
				if submatch != "" {
					snake += submatch + "_"
				}
			}
		}
		snake = strings.TrimSuffix(strings.TrimSuffix(snake, "."), "_") + "."
	}
	return strings.ToLower(strings.TrimSuffix(snake, "."))
}
