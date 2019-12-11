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
	memstatsPauseNsMetricPath       = "memstats.PauseNs.\\.*"
	memstatsPauseEndMetricPath      = "memstats.PauseEnd.\\.*"
	memstatsNumGCMetricPath         = "memstats.NumGC"
	memstatsBySizeSizeMetricPath    = "memstats.BySize.\\.*.Size"
	memstatsBySizeMallocsMetricPath = "memstats.BySize.\\.*.Mallocs"
	memstatsBySizeFreesMetricPath   = "memstats.BySize.\\.*.Frees"
	memstatsBySizeDimensionPath     = "memstats.BySize"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} {
		return &Monitor{
			keysFromPaths:    map[string][]string{},
			keysFromDimPaths: map[string][]string{},
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
	keysFromPaths    map[string][]string
	keysFromDimPaths map[string][]string
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
		if m.keysFromPaths[mConf.Name], err = utils.SplitString(mConf.JSONPath, []rune(mConf.PathSeparator)[0], escape); err != nil {
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
		dps[mConf.JSONPath] = make([]*datapoint.Datapoint, 0)
		for _, jsonKey := range m.findMatchingMapKeys(m.keysFromPaths[mConf.Name][0], jsonObj) {
			dp := datapoint.Datapoint{Dimensions: map[string]string{}}
			if applicationName != "" {
				dp.Dimensions["application_name"] = applicationName
			}
			m.addDps(dps, &dp, []string{jsonKey}, 0, jsonObj[jsonKey], mConf)
		}
	}
	return dps, nil
}

func (m *Monitor) findMatchingMapKeys(regex string, aMap map[string]interface{}) []string {
	if aMap[regex] != nil {
		return []string{regex}
	}
	var keys []string
	for key := range aMap {
		matched, err := regexp.MatchString(regex, key)
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

// addDps sets metric values and dimensions by traversing jsonValue recursively using keys derived by regex matching
// configured JSON paths. When addDps encounters and array type JSON value it clones the datapoint argument and adds
// dimension to hold the array index.
func (m *Monitor) addDps(dps map[string][]*datapoint.Datapoint, dp *datapoint.Datapoint, keys []string, keyFromPathIndex int, value interface{}, conf *MetricConfig) {
	keysFromPath, pathKeysLen, nextKeyFromPathIndex := m.keysFromPaths[conf.Name], len(m.keysFromPaths[conf.Name]), keyFromPathIndex+1
	atPathEnd := nextKeyFromPathIndex == pathKeysLen
	switch value := value.(type) {
	case map[string]interface{}:
		if atPathEnd {
			m.logger.Debugf("Expecting numeric value in path %s but found %+v. Configured path regex %s", joinWords(snakeCaseSlice(keys), conf.PathSeparator), value, conf.JSONPath)
			return
		}
		if len(value) == 0 {
			m.logger.Debugf("Empty object at %s before end of path. Configured path regex %s", joinWords(snakeCaseSlice(keys), conf.PathSeparator), conf.JSONPath)
			return
		}
		dpCopy, dims := dp, dp.Dimensions
		nextKeyFromPath := keysFromPath[nextKeyFromPathIndex]
		for i, nextKey := range m.findMatchingMapKeys(nextKeyFromPath, value) {
			if i > 0 {
				dpCopy = &datapoint.Datapoint{Dimensions: utils.CloneStringMap(dims)}
			}
			for _, dConf := range conf.DimensionConfigs {
				if keysFromDimPath := m.keysFromDimPaths[dConf.Name]; len(keysFromDimPath) != 0 && len(keysFromDimPath) == nextKeyFromPathIndex {
					dpCopy.Dimensions[dConf.Name] = nextKey
				}
			}
			keys := append(keys, nextKey)
			m.addDps(dps, dpCopy, append(keys[:0:0], keys...), nextKeyFromPathIndex, value[nextKey], conf)
		}
	case []interface{}:
		if atPathEnd {
			m.logger.Debugf("Expecting numeric value in path %s but found %+v. Configured path regex %s", joinWords(snakeCaseSlice(keys), conf.PathSeparator), value, conf.JSONPath)
			return
		}
		if len(value) == 0 {
			m.logger.Debugf("Empty object at %s before end of path. Configured path regex %s", joinWords(snakeCaseSlice(keys), conf.PathSeparator), conf.JSONPath)
			return
		}
		array, arrayIndexFromPath := value, keysFromPath[nextKeyFromPathIndex]
		arrayIndexFromPathCompiled, err := regexp.Compile(arrayIndexFromPath)
		if err != nil {
			m.logger.Error(err)
			return
		}
		dpCopy, dims := dp, dp.Dimensions
		for arrayIndex, arrayValue := range array {
			arrayIndexStr := strconv.Itoa(arrayIndex)
			if arrayIndexFromPathCompiled.MatchString(arrayIndexStr) {
				if arrayIndex > 0 {
					dpCopy = &datapoint.Datapoint{Dimensions: utils.CloneStringMap(dims)}
				}
				m.addArrayIndexDim(dpCopy.Dimensions, arrayIndexStr, keys, keyFromPathIndex, conf)
				m.addDps(dps, dpCopy, append(keys[:0:0], keys...), nextKeyFromPathIndex, arrayValue, conf)
			}
		}
	default:
		m.addDp(dps, dp, value, keys, atPathEnd, conf)
	}
}

func (m *Monitor) addDp(dps map[string][]*datapoint.Datapoint, dp *datapoint.Datapoint, value interface{}, keys []string, atPathEnd bool, conf *MetricConfig) {
	if !atPathEnd {
		m.logger.Debugf("Expecting object or array at %s before end of path. Found value %+v instead. Configured path regex %s", joinWords(snakeCaseSlice(keys), conf.PathSeparator), value, conf.JSONPath)
		return
	}
	if strings.TrimSpace(conf.Name) != "" {
		dp.Metric = conf.Name
	}
	if dp.Metric == "" {
		dp.Metric = joinWords(snakeCaseSlice(keys), ".")
	}
	dp.MetricType = conf.metricType()
	for _, dConf := range conf.DimensionConfigs {
		if strings.TrimSpace(dConf.Name) != "" && strings.TrimSpace(dConf.Value) != "" {
			dp.Dimensions[dConf.Name] = dConf.Value
		}
	}
	var err error
	if dp.Value, err = datapoint.CastMetricValueWithBool(value); err == nil {
		dps[conf.JSONPath] = append(dps[conf.JSONPath], dp)
	} else {
		m.logger.Debugf("Failed to set value for metric %s with JSON path %s because of type conversion error due to %+v", conf.Name, conf.JSONPath, err)
		m.logger.WithError(err).Error("Unable to set metric value")
	}
}

func (m *Monitor) addArrayIndexDim(dims map[string]string, arrayIndex string, keys []string, keyFromPathIndex int, metricConfig *MetricConfig) {
	nextKeyIndex := keyFromPathIndex + 1
	for _, conf := range metricConfig.DimensionConfigs {
		if keysFromDimPath := m.keysFromDimPaths[conf.Name]; len(keysFromDimPath) == nextKeyIndex {
			matched, err := regexp.MatchString(keysFromDimPath[keyFromPathIndex], keys[keyFromPathIndex])
			if err != nil {
				m.logger.Error(err)
				continue
			}
			if matched {
				dims[conf.Name] = arrayIndex
			}
			return
		}
	}
	if metricConfig.JSONPath == memstatsPauseNsMetricPath || metricConfig.JSONPath == memstatsPauseEndMetricPath {
		dims[metricConfig.JSONPath] = arrayIndex
		return
	}
	dims[joinWords(append(snakeCaseSlice(keys[:nextKeyIndex]), "index"), "_")] = arrayIndex
}

func (m *Monitor) preprocessDatapoint(dp *datapoint.Datapoint, metricPath string, mostRecentGCPauseIndex int64, now *time.Time) (bool, error) {
	if metricPath == memstatsPauseNsMetricPath || metricPath == memstatsPauseEndMetricPath {
		index, err := strconv.ParseInt(dp.Dimensions[metricPath], 10, 0)
		if err == nil && index == mostRecentGCPauseIndex {
			dp.Metric = memstatsMostRecentGcPauseNs
			if metricPath == memstatsPauseEndMetricPath {
				dp.Metric = memstatsMostRecentGcPauseEnd
			}
			// For index dimension key is equal to metric path for default metrics memstats.PauseNs and memstats.PauseEnd
			delete(dp.Dimensions, metricPath)
		} else {
			if err != nil {
				err = fmt.Errorf("failed to set metric GC pause. %+v", err)
			}
			return false, err
		}
	}
	// Renaming auto created dimension 'memstats.BySize' that stores array index to 'class'
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
