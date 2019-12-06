package expvar

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

const (
	memstatsPauseNsMetricPath       = "memstats/PauseNs/.*"
	memstatsPauseEndMetricPath      = "memstats/PauseEnd/.*"
	memstatsNumGCMetricPath         = "memstats/NumGC"
	memstatsBySizeSizeMetricPath    = "memstats/BySize/.*/Size"
	memstatsBySizeMallocsMetricPath = "memstats/BySize/.*/Mallocs"
	memstatsBySizeFreesMetricPath   = "memstats/BySize/.*/Frees"
	memstatsBySizeDimensionPath     = "memstats/BySize"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} {
		return &Monitor{
			metricsKeys:      map[string][]string{},
			dimsKeys:         map[string][]string{},
			allMetricConfigs: nil,
		}
	}, &Config{})
}

// Monitor for expvar metrics
type Monitor struct {
	Output           types.FilteringOutput
	cancel           context.CancelFunc
	ctx              context.Context
	client           *http.Client
	url              *url.URL
	runInterval      time.Duration
	metricsKeys      map[string][]string
	dimsKeys         map[string][]string
	allMetricConfigs []*MetricConfig
	logger           log.FieldLogger
}

// Configure monitor
func (m *Monitor) Configure(conf *Config) (err error) {
	m.logger = log.WithFields(log.Fields{"monitorType": monitorType})
	if m.Output.HasAnyExtraMetrics() {
		conf.EnhancedMetrics = true
	}
	m.allMetricConfigs = conf.getAllMetricConfigs()
	for _, mConf := range m.allMetricConfigs {
		if m.metricsKeys[mConf.Name], err = utils.SplitString(mConf.JSONPath, []rune(mConf.PathSeparator)[0], escape); err != nil {
			return err
		}
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
		dpsMap, err := m.fetchMetrics()
		if err != nil {
			m.logger.WithError(err).Error("could not get expvar metrics")
			return
		}
		mostRecentGCPauseIndex := getMostRecentGCPauseIndex(dpsMap)
		now := time.Now()
		for metricPath, dps := range dpsMap {
			// This is the filtering in place trick from https://github.com/golang/go/wiki/SliceTricks#filter-in-place
			n := 0
			for i := range dps {
				shouldSend, err := m.preprocessDatapoint(dps[i], metricPath, mostRecentGCPauseIndex, &now)
				if err != nil {
					m.logger.Error(err)
					continue
				}
				if shouldSend {
					dps[n] = dps[i]
					n++
				}
			}
			m.Output.SendDatapoints(dps[:n]...)
		}
	}, m.runInterval)
	return nil
}

func (m *Monitor) fetchMetrics() (map[string][]*datapoint.Datapoint, error) {
	resp, err := m.client.Get(m.url.String())
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	jsonObj := make(map[string]interface{})
	err = json.Unmarshal(body, &jsonObj)
	if err != nil {
		return nil, err
	}
	applicationName, err := getApplicationName(jsonObj)
	if err != nil {
		m.logger.Warn(err)
	}
	dps := make(map[string][]*datapoint.Datapoint)
	for _, mConf := range m.allMetricConfigs {
		dp := datapoint.Datapoint{Dimensions: map[string]string{}}
		if applicationName != "" {
			dp.Dimensions["application_name"] = applicationName
		}
		dps[mConf.JSONPath] = make([]*datapoint.Datapoint, 0)
		for _, jsonKey := range m.findKeys(m.metricsKeys[mConf.Name][0], jsonObj) {
			m.setDps(dps, &dp, jsonKey, 0, jsonObj[jsonKey], mConf)
		}
	}
	return dps, nil
}

func (m *Monitor) findKeys(pattern string, aMap map[string]interface{}) []string {
	if aMap[pattern] != nil {
		return []string{pattern}
	}
	var keys []string
	for key := range aMap {
		matched, err := regexp.MatchString(pattern, key)
		if err != nil {
			m.logger.Error(err)
			continue
		}
		if matched {
			keys = append(keys, key)
		}
	}
	return keys
}

// setDps sets metric values and dimensions by traversing jsonValue recursively using keys derived by regex matching
// configured JSON paths. When setDps encounters and array type JSON value it clones the datapoint argument and adds
// dimension to hold the array index.
func (m *Monitor) setDps(dps map[string][]*datapoint.Datapoint, dp *datapoint.Datapoint, jsonKey string, jsonKeyPatternIndex int, jsonValue interface{}, metricConfig *MetricConfig) {
	dp.Metric = joinWords(snakeCaseSlice([]string{dp.Metric, jsonKey}), ".")
	jsonKeyPatterns, nextJSONKeyPatternIndex := m.metricsKeys[metricConfig.Name], jsonKeyPatternIndex+1
	if jsonKeyPatternIndex >= len(jsonKeyPatterns) {
		m.logger.Errorf("failed to find metric value in path: %s", metricConfig.JSONPath)
		return
	}
	switch jsonValue := jsonValue.(type) {
	case map[string]interface{}:
		nextJSONKeyPattern := jsonKeyPatterns[nextJSONKeyPatternIndex]
		dpCopy, metric, dims := dp, dp.Metric, dp.Dimensions
		for i, nextJSONKey := range m.findKeys(nextJSONKeyPattern, jsonValue) {
			if i > 0 {
				dpCopy = &datapoint.Datapoint{Metric: metric, Dimensions: utils.CloneStringMap(dims)}
			}
			for _, dConf := range metricConfig.DimensionConfigs {
				if jsonKeyPatternsDim := m.dimsKeys[dConf.Name]; len(jsonKeyPatternsDim) != 0 && len(jsonKeyPatternsDim) == nextJSONKeyPatternIndex {
					dpCopy.Dimensions[dConf.Name] = nextJSONKey
				}
			}
			m.setDps(dps, dpCopy, nextJSONKey, nextJSONKeyPatternIndex, jsonValue[nextJSONKey], metricConfig)
		}
	case []interface{}:
		jsonArray, arrayIndexPattern := jsonValue, jsonKeyPatterns[nextJSONKeyPatternIndex]
		arrayIndexPatternRegexp, err := regexp.Compile(arrayIndexPattern)
		if err != nil {
			m.logger.Error(err)
			return
		}
		dpCopy, metric, dims := dp, dp.Metric, dp.Dimensions
		for jsonArrayIndex, jsonArrayValue := range jsonArray {
			jsonArrayIndexStr := strconv.Itoa(jsonArrayIndex)
			if arrayIndexPatternRegexp.MatchString(jsonArrayIndexStr) {
				if jsonArrayIndex > 0 {
					dpCopy = &datapoint.Datapoint{Metric: metric, Dimensions: utils.CloneStringMap(dims)}
				}
				m.setIndexDim(dpCopy.Dimensions, jsonArrayIndexStr, jsonKey, jsonKeyPatternIndex, metricConfig)
				m.setDps(dps, dpCopy, "", nextJSONKeyPatternIndex, jsonArrayValue, metricConfig)
			}
		}
	default:
		if strings.TrimSpace(metricConfig.Name) != "" {
			dp.Metric = metricConfig.Name
		}
		dp.MetricType = metricConfig.metricType()
		for _, dConf := range metricConfig.DimensionConfigs {
			if strings.TrimSpace(dConf.Name) != "" && strings.TrimSpace(dConf.Value) != "" {
				dp.Dimensions[dConf.Name] = dConf.Value
			}
		}
		var err error
		if dp.Value, err = datapoint.CastMetricValueWithBool(jsonValue); err == nil {
			dps[metricConfig.JSONPath] = append(dps[metricConfig.JSONPath], dp)
		} else {
			m.logger.Debugf("Failed to set value for metric %s with JSON path %s because of type conversion error due to %+v", metricConfig.Name, metricConfig.JSONPath, err)
			m.logger.WithError(err).Error("Unable to set metric value")
			return
		}
	}
}

func (m *Monitor) setIndexDim(dims map[string]string, jsonArrayIndex string, jsonKey string, jsonKeyIndex int, metricConfig *MetricConfig) {
	jsonKeyPatterns, nextJSONKeyIndex := m.metricsKeys[metricConfig.Name], jsonKeyIndex+1
	for _, conf := range metricConfig.DimensionConfigs {
		if jsonKeyPatternsDim := m.dimsKeys[conf.Name]; len(jsonKeyPatternsDim) == nextJSONKeyIndex {
			matched, err := regexp.MatchString(jsonKeyPatternsDim[jsonKeyIndex], jsonKey)
			if err != nil {
				m.logger.Error(err)
				continue
			}
			if matched {
				dims[conf.Name] = jsonArrayIndex
			}
			return
		}
	}
	if metricConfig.JSONPath == memstatsPauseNsMetricPath || metricConfig.JSONPath == memstatsPauseEndMetricPath {
		dims[metricConfig.JSONPath] = jsonArrayIndex
		return
	}
	dims[joinWords(append(snakeCaseSlice(jsonKeyPatterns[:nextJSONKeyIndex]), "index"), "_")] = jsonArrayIndex
}

func (m *Monitor) preprocessDatapoint(dp *datapoint.Datapoint, metricPath string, mostRecentGCPauseIndex int64, now *time.Time) (bool, error) {
	if metricPath == memstatsPauseNsMetricPath || metricPath == memstatsPauseEndMetricPath {
		index, err := strconv.ParseInt(dp.Dimensions[metricPath], 10, 0)
		if err == nil && index == mostRecentGCPauseIndex {
			dp.Metric = memstatsMostRecentGcPauseNs
			if metricPath == memstatsPauseEndMetricPath {
				dp.Metric = memstatsMostRecentGcPauseEnd
			}
			// For index dimension key is equal to metric path for default metrics memstats/PauseNs and memstats/PauseEnd
			delete(dp.Dimensions, metricPath)
		} else {
			if err != nil {
				err = fmt.Errorf("failed to set metric GC pause. %+v", err)
			}
			return false, err
		}
	}
	// Renaming auto created dimension 'memstats/BySize' that stores array index to 'class'
	if metricPath == memstatsBySizeSizeMetricPath || metricPath == memstatsBySizeMallocsMetricPath || metricPath == memstatsBySizeFreesMetricPath {
		dp.Dimensions["class"] = dp.Dimensions[memstatsBySizeDimensionPath]
		delete(dp.Dimensions, memstatsBySizeDimensionPath)
	}
	dp.Timestamp = *now
	return true, nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
