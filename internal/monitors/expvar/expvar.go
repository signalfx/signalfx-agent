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

const (
	memstatsPauseNsMetricPath       = "memstats.PauseNs"
	memstatsPauseEndMetricPath      = "memstats.PauseEnd"
	memstatsNumGCMetricPath         = "memstats.NumGC"
	memstatsBySizeSizeMetricPath    = "memstats.BySize.Size"
	memstatsBySizeMallocsMetricPath = "memstats.BySize.Mallocs"
	memstatsBySizeFreesMetricPath   = "memstats.BySize.Frees"
	memstatsBySizeDimensionPath     = "memstats.BySize"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} {
		return &Monitor{
			metricPathsParts:    map[string][]string{},
			dimensionPathsParts: map[DimensionConfig][]string{},
			allMetricConfigs:    nil,
		}
	}, &Config{})
}

// Monitor for expvar metrics
type Monitor struct {
	Output              types.FilteringOutput
	cancel              context.CancelFunc
	ctx                 context.Context
	client              *http.Client
	url                 *url.URL
	runInterval         time.Duration
	metricPathsParts    map[string][]string
	dimensionPathsParts map[DimensionConfig][]string
	allMetricConfigs    []MetricConfig
	logger              log.FieldLogger
}

// Configure monitor
func (m *Monitor) Configure(conf *Config) error {
	m.logger = log.WithFields(log.Fields{"monitorType": monitorType})

	if m.Output.HasAnyExtraMetrics() {
		conf.EnhancedMetrics = true
	}

	m.allMetricConfigs = conf.getAllMetricConfigs()

	for _, mConf := range m.allMetricConfigs {
		m.metricPathsParts[mConf.name()] = strings.Split(mConf.JSONPath, ".")
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
			for _, dp := range dps {
				if err := m.sendDatapoint(dp, metricPath, mostRecentGCPauseIndex, &now); err != nil {
					m.logger.Error(err)
				}
			}
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
	metricsJSON := make(map[string]interface{})
	err = json.Unmarshal(body, &metricsJSON)
	if err != nil {
		return nil, err
	}
	applicationName, err := getApplicationName(metricsJSON)
	if err != nil {
		m.logger.Warn(err)
	}
	dpsMap := make(map[string][]*datapoint.Datapoint)
	for i := range m.allMetricConfigs {
		mConf := m.allMetricConfigs[i]
		dp := datapoint.Datapoint{Dimensions: map[string]string{}}
		if applicationName != "" {
			dp.Dimensions["application_name"] = applicationName
		}
		dpsMap[mConf.JSONPath] = make([]*datapoint.Datapoint, 0)
		m.setDatapoints(metricsJSON[m.metricPathsParts[mConf.name()][0]], &mConf, &dp, dpsMap, 0)
	}

	return dpsMap, nil
}

// setDatapoints the dp argument should be a pointer to a zero value datapoint and
// traverses v recursively following metric path parts in mConf.keys[]
// adds dimensions along the way and sets metric value in the end
// clones datapoints and add array index dimension for array values in v
func (m *Monitor) setDatapoints(v interface{}, mc *MetricConfig, dp *datapoint.Datapoint, dpsMap map[string][]*datapoint.Datapoint, metricPathIndex int) {
	if metricPathIndex >= len(m.metricPathsParts[mc.name()]) {
		m.logger.Errorf("failed to find metric value in path: %s", mc.JSONPath)
		return
	}
	switch set := v.(type) {
	case map[string]interface{}:
		for _, dConf := range mc.DimensionConfigs {
			if len(m.dimensionPathsParts[dConf]) != 0 && len(m.dimensionPathsParts[dConf]) == metricPathIndex {
				dp.Dimensions[dConf.Name] = m.metricPathsParts[mc.name()][metricPathIndex]
			}
		}
		m.setDatapoints(set[m.metricPathsParts[mc.name()][metricPathIndex+1]], mc, dp, dpsMap, metricPathIndex+1)
	case []interface{}:
		clone := dp
		for index, value := range set {
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
				clone.Dimensions[strings.Join(m.metricPathsParts[mc.name()][:metricPathIndex+1], ".")] = fmt.Sprint(index)
			}
			m.setDatapoints(value, mc, clone, dpsMap, metricPathIndex)
		}
	default:
		dp.Metric, dp.MetricType = mc.name(), mc.metricType()
		for _, dConf := range mc.DimensionConfigs {
			if strings.TrimSpace(dConf.Name) != "" && strings.TrimSpace(dConf.Value) != "" {
				dp.Dimensions[dConf.Name] = dConf.Value
			}
		}
		var err error
		if dp.Value, err = datapoint.CastMetricValueWithBool(v); err == nil {
			dpsMap[mc.JSONPath] = append(dpsMap[mc.JSONPath], dp)
		} else {
			m.logger.Debugf("failed to set value for metric %s with JSON path %s because of type conversion error due to %+v", mc.name(), mc.JSONPath, err)
			m.logger.WithError(err).Error("Unable to set metric value")
			return
		}
	}
}

func (m *Monitor) sendDatapoint(dp *datapoint.Datapoint, metricPath string, mostRecentGCPauseIndex int64, now *time.Time) error {
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
