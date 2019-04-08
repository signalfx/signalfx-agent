package expvar

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

const monitorType = "expvar"

const (
	gauge                                  = "gauge"
	cumulative                             = "cumulative"
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

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Monitor for expvar metrics
type Monitor struct {
	Output              types.Output
	cancel              context.CancelFunc
	ctx                 context.Context
	client              *http.Client
	url                 *url.URL
	runInterval         time.Duration
	metricTypes         map[*MetricConfig]datapoint.MetricType
	metricPathsParts    map[*MetricConfig][]string
	dimensionPathsParts map[*DimensionConfig][]string
	allMetricConfigs    []*MetricConfig
}

// Configure monitor
func (m *Monitor) Configure(conf *Config) error {
	m.metricTypes = map[*MetricConfig]datapoint.MetricType{}
	m.metricPathsParts = map[*MetricConfig][]string{}
	m.dimensionPathsParts = map[*DimensionConfig][]string{}
	m.allMetricConfigs = []*MetricConfig{}
	m.addDefaultMetricConfigs(conf.EnhancedMetrics)
	for _, mConf := range conf.MetricConfigs {
		if m.metricTypes[mConf] = datapoint.Gauge; strings.TrimSpace(strings.ToLower(mConf.Type)) == cumulative {
			m.metricTypes[mConf] = datapoint.Counter
		}
		m.metricPathsParts[mConf] = strings.Split(mConf.JSONPath, ".")
		m.allMetricConfigs = append(m.allMetricConfigs, mConf)
	}
	m.url = &url.URL{
		Scheme: func() string {
			if conf.UseHTTPS {
				return "https"
			}
			return "http"
		}(),
		Host: fmt.Sprintf("%s:%d", conf.Host, conf.Port),
		Path: conf.Path,
	}
	m.runInterval = time.Duration(conf.IntervalSeconds) * time.Second
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.client = &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 10,
		},
		Timeout: 300 * time.Millisecond,
	}
	utils.RunOnInterval(m.ctx, func() {
		dpsMap, err := m.fetchMetrics(conf)
		if err != nil {
			logger.WithError(err).Error("could not get expvar metrics")
			return
		}
		mostRecentGCPauseIndex := getMostRecentGCPauseIndex(dpsMap)
		now := time.Now()
		for metricPath, dps := range dpsMap {
			for _, dp := range dps {
				if err := m.sendDatapoint(conf, dp, metricPath, mostRecentGCPauseIndex, &now); err != nil {
					logger.Error(err)
				}
			}
		}
	}, m.runInterval)
	return nil
}

func (m *Monitor) addDefaultMetricConfigs(enhancedMetrics bool) {
	memstatsMetricPathsGauge := []string{
		"memstats.HeapAlloc", "memstats.HeapSys", "memstats.HeapIdle", "memstats.HeapInuse", "memstats.HeapReleased",
		"memstats.HeapObjects", "memstats.StackInuse", "memstats.StackSys", "memstats.MSpanInuse", "memstats.MSpanSys",
		"memstats.MCacheInuse", "memstats.MCacheSys", "memstats.BuckHashSys", "memstats.GCSys", "memstats.OtherSys",
		"memstats.Sys", "memstats.NextGC", "memstats.LastGC", "memstats.GCCPUFraction", "memstats.EnableGC",
		"memstats.DebugGC", memstatsPauseNsMetricPath, memstatsPauseEndMetricPath,
	}
	memstatsMetricPathsCumulative := []string{
		"memstats.TotalAlloc", "memstats.Lookups", "memstats.Mallocs", "memstats.Frees", "memstats.PauseTotalNs",
		memstatsNumGCMetricPath, "memstats.NumForcedGC",
	}
	if enhancedMetrics {
		memstatsMetricPathsGauge = append(memstatsMetricPathsGauge, "memstats.Alloc")
		memstatsMetricPathsCumulative = append(memstatsMetricPathsCumulative, memstatsBySizeSizeMetricPath, memstatsBySizeMallocsMetricPath, memstatsBySizeFreesMetricPath)
	}
	for _, path := range memstatsMetricPathsGauge {
		mConf := &MetricConfig{Name: toSnakeCase(path), JSONPath: path, Type: gauge, DimensionConfigs: []*DimensionConfig{{}}}
		m.metricTypes[mConf] = datapoint.Gauge
		m.metricPathsParts[mConf] = strings.Split(path, ".")
		m.allMetricConfigs = append(m.allMetricConfigs, mConf)
	}
	for _, path := range memstatsMetricPathsCumulative {
		mConf := &MetricConfig{Name: toSnakeCase(path), JSONPath: path, Type: cumulative, DimensionConfigs: []*DimensionConfig{{}}}
		m.metricTypes[mConf] = datapoint.Counter
		m.metricPathsParts[mConf] = strings.Split(path, ".")
		m.allMetricConfigs = append(m.allMetricConfigs, mConf)
	}
}

func (m *Monitor) fetchMetrics(conf *Config) (map[string][]*datapoint.Datapoint, error) {
	resp, err := m.client.Get(m.url.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	metricsJSON := make(map[string]interface{})
	err = json.Unmarshal(body, &metricsJSON)
	if err != nil {
		return nil, err
	}
	applicationName, err := getApplicationName(metricsJSON)
	if err != nil {
		logger.Warn(err)
	}
	dpsMap := make(map[string][]*datapoint.Datapoint)
	for _, mConf := range m.allMetricConfigs {
		dp := datapoint.Datapoint{Dimensions: map[string]string{}}
		if applicationName != "" {
			dp.Dimensions["application_name"] = applicationName
		}
		dpsMap[mConf.JSONPath] = make([]*datapoint.Datapoint, 0)
		m.setDatapoints(metricsJSON[m.metricPathsParts[mConf][0]], mConf, &dp, dpsMap, 0)
	}

	return dpsMap, nil
}

// setDatapoints the dp argument should be a pointer to a zero value datapoint
// traverses v recursively following metric path parts in mConf.keys[]
// adds dimensions along the way and sets metric value in the end
// clones datapoints and add array index dimension for array values in v
func (m *Monitor) setDatapoints(v interface{}, mc *MetricConfig, dp *datapoint.Datapoint, dpsMap map[string][]*datapoint.Datapoint, metricPathIndex int) {
	if metricPathIndex >= len(m.metricPathsParts[mc]) {
		logger.Errorf("failed to find metric value in path: %s", mc.JSONPath)
		return
	}
	switch v.(type) {
	case nil:
		logger.Errorf("failed to find value %s with JSON path %s", mc.name(), mc.JSONPath)
		return
	case map[string]interface{}:
		for _, dConf := range mc.DimensionConfigs {
			if len(m.dimensionPathsParts[dConf]) != 0 && len(m.dimensionPathsParts[dConf]) == metricPathIndex {
				dp.Dimensions[dConf.Name] = m.metricPathsParts[mc][metricPathIndex]
			}
		}
		set := v.(map[string]interface{})
		m.setDatapoints(set[m.metricPathsParts[mc][metricPathIndex+1]], mc, dp, dpsMap, metricPathIndex+1)
	case []interface{}:
		values := v.([]interface{})
		clone := dp
		for index, value := range values {
			if index > 0 {
				clone = &datapoint.Datapoint{Dimensions: utils.CloneStringMap(clone.Dimensions)}
			}
			createIndexDimension := true
			for _, conf := range mc.DimensionConfigs {
				if len(m.dimensionPathsParts[conf]) == metricPathIndex+1 {
					clone.Dimensions[conf.Name] = fmt.Sprint(index)
					createIndexDimension = false
				}
			}
			if createIndexDimension {
				clone.Dimensions[strings.Join(m.metricPathsParts[mc][:metricPathIndex+1], ".")] = fmt.Sprint(index)
			}
			m.setDatapoints(value, mc, clone, dpsMap, metricPathIndex)
		}
	default:
		dp.Metric, dp.MetricType = mc.name(), m.metricTypes[mc]
		for _, dConf := range mc.DimensionConfigs {
			if strings.TrimSpace(dConf.Name) != "" && strings.TrimSpace(dConf.Value) != "" {
				dp.Dimensions[dConf.Name] = dConf.Value
			}
		}
		var err error
		if dp.Value, err = datapoint.CastMetricValueWithBool(v); err == nil {
			dpsMap[mc.JSONPath] = append(dpsMap[mc.JSONPath], dp)
		} else {
			logger.Debugf("failed to set value for metric %s with JSON path %s because of type conversion error due to %+v", mc.name(), mc.JSONPath, err)
			logger.WithError(err).Error("Unable to set metric value")
			return
		}
	}
}

func (m *Monitor) sendDatapoint(conf *Config, dp *datapoint.Datapoint, metricPath string, mostRecentGCPauseIndex int64, now *time.Time) error {
	if metricPath == memstatsPauseNsMetricPath || metricPath == memstatsPauseEndMetricPath {
		index, err := strconv.ParseInt(dp.Dimensions[metricPath], 10, 0)
		if err == nil && index == mostRecentGCPauseIndex {
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
	// Renaming auto created dimension 'memstats.BySize' that stores array index to 'class'
	if metricPath == memstatsBySizeSizeMetricPath || metricPath == memstatsBySizeMallocsMetricPath || metricPath == memstatsBySizeFreesMetricPath {
		dp.Dimensions["class"] = dp.Dimensions[memstatsBySizeDimensionPath]
		delete(dp.Dimensions, memstatsBySizeDimensionPath)
	}
	dp.Timestamp = *now
	m.Output.SendDatapoint(dp)
	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
