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

// Config for this monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	// Host of the expvar endpoint
	Host string `yaml:"host" validate:"required"`
	// Port of the expvar endpoint
	Port uint16 `yaml:"port" validate:"required"`
	// If true, the agent will connect to the host using HTTPS instead of plain HTTP.
	UseHTTPS bool `yaml:"useHTTPS"`
	// If useHTTPS is true and this option is also true, the host's TLS cert will not be verified.
	SkipVerify bool `yaml:"skipVerify"`
	// Path to the expvar endpoint, usually `/debug/vars` (the default).
	Path string `yaml:"path" default:"/debug/vars"`
	// Excludes metrics memstats.alloc, memstats.by_size.size, memstats.by_size.mallocs and memstats.by_size.frees if false
	EnhancedMetrics bool `yaml:"enhancedMetrics"`
	// Metrics configurations
	MetricConfigs []*MetricConfig `yaml:"metrics"`
	url           *url2.URL
	runInterval   time.Duration
}

// MetricConfig for metric
type MetricConfig struct {
	// Metric name
	Name string `yaml:"name"`
	// JSON path of the metric value
	JSONPath string `yaml:"JSONPath" validate:"required"`
	// SignalFx metric type. Possible values are "gauge" or "counter"
	Type string `yaml:"type" validate:"required"`
	// Metric dimensions
	DimensionConfigs []*DimensionConfig `yaml:"dimensions"`
	metricType       datapoint.MetricType
	// Slice of Path substrings separated by .
	keys []string
}

// DimensionConfig for metric dimension
type DimensionConfig struct {
	// Dimension name
	Name string `yaml:"name"`
	// JSON path of the dimension value
	JSONPath string `yaml:"JSONPath"`
	// Dimension value
	Value string `yaml:"value"`
	// Slice of JSONPath substrings separated by .
	splits []string
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
	if conf.MetricConfigs != nil {
		for _, mConf := range conf.MetricConfigs {
			_type := strings.TrimSpace(strings.ToLower(mConf.Type))
			if _type != gauge && _type != counter {
				return fmt.Errorf("unsupported metric type %s. Supported metric types are: %s, %s", mConf.Type, gauge, counter)
			}
			for _, dConf := range mConf.DimensionConfigs {
				if dConf != nil && strings.TrimSpace(dConf.JSONPath) != "" && !strings.HasPrefix(mConf.JSONPath, dConf.JSONPath) {
					return fmt.Errorf("invalid dimension path %s. Dimension path not parent path of metric path %s", dConf.JSONPath, mConf.JSONPath)
				}
			}
		}
	}
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
	memstatsMetricPathsGauge := []string{"memstats.HeapAlloc", "memstats.HeapSys", "memstats.HeapIdle", "memstats.HeapInuse", "memstats.HeapReleased", "memstats.HeapObjects", "memstats.StackInuse", "memstats.StackSys", "memstats.MSpanInuse", "memstats.MSpanSys", "memstats.MCacheInuse", "memstats.MCacheSys", "memstats.BuckHashSys", "memstats.GCSys", "memstats.OtherSys", "memstats.Sys", "memstats.NextGC", "memstats.LastGC", "memstats.GCCPUFraction", "memstats.EnableGC", "memstats.DebugGC", memstatsPauseNsMetricPath, memstatsPauseEndMetricPath}
	memstatsMetricPathsCounter := []string{"memstats.TotalAlloc", "memstats.Lookups", "memstats.Mallocs", "memstats.Frees", "memstats.PauseTotalNs", memstatsNumGCMetricPath, "memstats.NumForcedGC"}

	if conf.EnhancedMetrics {
		memstatsMetricPathsGauge = append(memstatsMetricPathsGauge, "memstats.Alloc")
		memstatsMetricPathsCounter = append(memstatsMetricPathsCounter, memstatsBySizeSizeMetricPath, memstatsBySizeMallocsMetricPath, memstatsBySizeFreesMetricPath)
	}
	if conf.MetricConfigs == nil {
		conf.MetricConfigs = make([]*MetricConfig, 0, len(memstatsMetricPathsGauge)+len(memstatsMetricPathsCounter))
	}
	// Initializing custom metrics
	for _, mConf := range conf.MetricConfigs {
		if strings.TrimSpace(mConf.Name) == "" {
			mConf.Name = toSnakeCase(mConf.JSONPath)
		}
		mConf.metricType = func() datapoint.MetricType {
			if strings.TrimSpace(strings.ToLower(mConf.Type)) == gauge {
				return datapoint.Gauge
			}
			return datapoint.Counter
		}()
		if mConf.DimensionConfigs == nil {
			mConf.DimensionConfigs = []*DimensionConfig{{}}
		}
		mConf.keys = strings.Split(mConf.JSONPath, ".")
	}
	// Initializing default metrics
	for _, path := range memstatsMetricPathsGauge {
		conf.MetricConfigs = append(conf.MetricConfigs, &MetricConfig{Name: toSnakeCase(path), JSONPath: path, Type: gauge, metricType: datapoint.Gauge, DimensionConfigs: []*DimensionConfig{{}}, keys: strings.Split(path, ".")})
	}
	for _, path := range memstatsMetricPathsCounter {
		conf.MetricConfigs = append(conf.MetricConfigs, &MetricConfig{Name: toSnakeCase(path), JSONPath: path, Type: counter, metricType: datapoint.Counter, DimensionConfigs: []*DimensionConfig{{}}, keys: strings.Split(path, ".")})
	}
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
			// For index dimension key is equal to metric path for default metrics memstats.PauseNs and memstats.PauseEnd
			delete(dp.Dimensions, metricPath)
		} else {
			if err != nil {
				err = fmt.Errorf("failed to set metric most recent GC pause metric. %+v", err)
			}
			return err
		}
	}
	// Renaming auto created dimension 'memstats.BySize' to 'class'
	if metricPath == memstatsBySizeSizeMetricPath || metricPath == memstatsBySizeMallocsMetricPath || metricPath == memstatsBySizeFreesMetricPath {
		dp.Dimensions["class"] = dp.Dimensions[memstatsBySizeDimensionPath]
		delete(dp.Dimensions, memstatsBySizeDimensionPath)
	}
	dp.Timestamp = *now
	dp.Dimensions["application_host"] = conf.Host
	dp.Dimensions["application_expvar_port"] = strconv.FormatInt(int64(conf.Port), 10)
	dp.Dimensions["plugin"] = monitorType
	m.Output.SendDatapoint(dp)
	return nil
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
	var applicationName string
	if _map, ok := metricsJSON.(map[string]interface{}); ok {
		if applicationName, err = getApplicationName(_map); err != nil {
			logger.Error(err)
		}
		dpsMap = make(map[string][]*datapoint.Datapoint)
		for _, _metric := range conf.MetricConfigs {
			dp := datapoint.Datapoint{Dimensions: map[string]string{}}
			if applicationName != "" {
				dp.Dimensions["application_name"] = applicationName
			}
			dpsMap[_metric.JSONPath] = make([]*datapoint.Datapoint, 0)
			setDatapoints(_map[_metric.keys[0]], _metric, &dp, &dpsMap, 0)
		}
	}
	return &dpsMap, nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

func (mConf *MetricConfig) dimensions() map[string]string {
	var dimensions map[string]string
	if len(mConf.DimensionConfigs) > 0 {
		dimensions = make(map[string]string, len(mConf.DimensionConfigs))
		for _, dConf := range mConf.DimensionConfigs {
			if strings.TrimSpace(dConf.Name) != "" && strings.TrimSpace(dConf.Value) != "" {
				dimensions[dConf.Name] = dConf.Value
			}
		}
	}
	return dimensions
}

// setDatapoints the dp argument should be a pointer to a zero value datapoint
// setDatapoints traverses v recursively following metric path parts in mConf.keys[]
// setDatapoints adds dimensions along the way and sets metric value in the end
// setDatapoints clones datapoints and add array index dimension for array values in v
func setDatapoints(v interface{}, mConf *MetricConfig, dp *datapoint.Datapoint, dpsMap *map[string][]*datapoint.Datapoint, i int) {
	if _map, ok := v.(map[string]interface{}); ok {
		for _, dConf := range mConf.DimensionConfigs {
			if len(dConf.splits) != 0 && len(dConf.splits) == i {
				dp.Dimensions[dConf.Name] = mConf.keys[i]
			}
		}
		setDatapoints(_map[mConf.keys[i+1]], mConf, dp, dpsMap, i+1)
	} else if arr, ok := v.([]interface{}); ok {
		_dp := dp
		for _i, _v := range arr {
			if _i > 0 {
				// At this point nothing is set for the do except possibly some dimensions
				_dp = &datapoint.Datapoint{Dimensions: copyDims(dp)}
			}
			makeNewIndexDimension := true
			for _, dConf := range mConf.DimensionConfigs {
				if len(dConf.splits) == i+1 {
					_dp.Dimensions[dConf.Name] = strconv.Itoa(_i)
					makeNewIndexDimension = false
				}
			}
			if makeNewIndexDimension {
				_dp.Dimensions[strings.Join(mConf.keys[:(i+1)], ".")] = strconv.Itoa(_i)
			}
			setDatapoints(_v, mConf, _dp, dpsMap, i)
		}
	} else {
		if v == nil {
			logger.Errorf("failed to find value for metric %s with JSON path %s", mConf.Name, mConf.JSONPath)
			return
		}
		dp.Metric, dp.MetricType = mConf.Name, mConf.metricType
		for _, dConf := range mConf.DimensionConfigs {
			if strings.TrimSpace(dConf.Name) != "" && strings.TrimSpace(dConf.Value) != "" {
				dp.Dimensions[dConf.Name] = dConf.Value
			}
		}
		var err error
		if dp.Value, err = datapoint.CastMetricValueWithBool(v); err == nil {
			(*dpsMap)[mConf.JSONPath] = append((*dpsMap)[mConf.JSONPath], dp)
		} else {
			logger.Errorf("failed to set value for metric %s with JSON path %s because of type conversion error due to %+v", mConf.Name, mConf.JSONPath, err)
			return
		}
	}
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

func getApplicationName(_map map[string]interface{}) (string, error) {
	if cmdline, ok := _map["cmdline"].([]interface{}); ok && len(cmdline) > 0 {
		if applicationName := strings.TrimSpace(slashLastRegexp.FindStringSubmatch(cmdline[0].(string))[0]); applicationName != "" {
			return applicationName, nil
		}
	}
	return "", fmt.Errorf("failed to get application name from the first array value of cmdline in map: %+v", _map)
}

func copyDims(dp *datapoint.Datapoint) map[string]string {
	dimensions := map[string]string{}
	for k, v := range dp.Dimensions {
		dimensions[k] = v
	}
	return dimensions
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
