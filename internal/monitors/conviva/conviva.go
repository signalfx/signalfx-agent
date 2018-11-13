package conviva

import (
	"context"
	"fmt"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/signalfx-agent/internal/core/common/dpmeta"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
	"math"
	"net/http"
	"strings"
	"time"
)

const monitorType = "conviva"

// MONITOR(conviva): This monitor uses version 2.4 of the Conviva Experience Insights REST APIs to pull
// `Real-Time/Live` video playing experience metrics from Conviva.
//
// Only `Live` conviva metrics listed
// [here](https://community.conviva.com/site/global/apis_data/experience_insights_api/index.gsp#metrics)
// are supported. All metrics are gauges. The Conviva metrics are converted to SignalFx metrics with dimensions
// named account and filter. The account dimension is the name of the Conviva account and the filter dimension
// is the name of the Conviva filter applied to the metric. In the case of MetricLenses, the constituent
// MetricLens metrics and MetricLens dimensions are included. The values of the MetricLens dimensions are
// derived from the values of the associated MetricLens dimension entities.
//
// Below is a sample YAML configuration showing the most basic configuration of the Conviva monitor
// using only the required fields. For this configuration the monitor will default to fetching quality MetricLens
// metrics for all dimensions from the default Conviva account using the `All Traffic` filter.
//
// ```
//monitors:
//- type: conviva
//  pulseUsername: <username>
//  pulsePassword: <password>
// ```
//
// Individual metrics are configured as a list of metricConfigs as shown in sample configuration below. The
// metrics a fetched using the specified metricParameter. Find the list of metric parameters
// [here](https://github.com/signalfx/integrations/blob/master/conviva/docs/conviva_metrics.md).
// The Conviva metrics reported to SignalFx are prefixed by `conviva.`, `conviva.quality_metriclens.` and
// `conviva.audience_metriclens.` accordingly. The metric names are the `titles` of the metrics
// [here](https://github.com/signalfx/integrations/tree/master/conviva/docs) which correspond to the Conviva
// `metric parameters` [here](https://community.conviva.com/site/global/apis_data/experience_insights_api/index.gsp#metrics).
// Where an account is not provided the default account is fetched and used. Where no filters are specified the
// `All Traffic` filter is used. Where MetricLens dimensions are not specified all MetricLens dimensions
// are fetched and used. The `_ALL_` keyword means all. MetricLens dimension configuration applies only to MetricLenses.
// If specified for a regular metric they will be ignored. MetricLens dimensions listed in `excludeMetricLensDimensions`
// will be excluded.
//
// ```
//monitors:
//- type: conviva
//  pulseUsername: <username>
//  pulsePassword: <password>
//  metricConfigs:
//    - account: c3.NBC
//      metricParameter: quality_metriclens
//      filters:
//        - All Traffic
//      metricLensDimensions:
//        - Cities
//    - metricParameter: avg_bitrate
//      filters:
//        - _ALL_
//    - metricParameter: concurrent_plays
//    - metricParameter: audience_metriclens
//      filters:
//        - All Traffic
//      metricLensDimensions:
//        - _ALL_
//      excludeMetricLensDimensions:
//        - CDNs
// ```
//
// Add the extra dimension metric_source as shown in sample configuration below for the convenience of searching
// for your metrics in SignalFx using the metric_source value you specify.
//
// ```
//monitors:
//- type: conviva
//  pulseUsername: <username>
//  pulsePassword: <password>
//  extraDimensions:
//    metric_source: conviva
// ```

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

const (
	metricURLFormat     = "https://api.conviva.com/insights/2.4/metrics.json?metrics=%s&account=%s&filter_ids=%s"
	metricLensURLFormat = metricURLFormat + "&metriclens_dimension_id=%d"
)

// Config for this monitor
type Config struct {
	config.MonitorConfig
	// Conviva Pulse username required with each API request.
	Username string `yaml:"pulseUsername" validate:"required"`
	// Conviva Pulse password required with each API request.
	Password       string `yaml:"pulsePassword" validate:"required" neverLog:"true"`
	TimeoutSeconds int    `yaml:"timeoutSeconds" default:"10"`
	// Conviva metrics to fetch. The default is quality_metriclens metric with the "All Traffic" filter applied and all quality_metriclens dimensions.
	MetricConfigs []*metricConfig `yaml:"metricConfigs"`
}

// Monitor for conviva metrics
type Monitor struct {
	Output  types.Output
	cancel  context.CancelFunc
	ctx     context.Context
	client  httpClient
	timeout time.Duration
}

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Configure monitor
func (m *Monitor) Configure(conf *Config) error {
	if conf.MetricConfigs == nil {
		conf.MetricConfigs = []*metricConfig{{MetricParameter: "quality_metriclens"}}
	}
	m.timeout = time.Duration(conf.TimeoutSeconds) * time.Second
	m.client = newConvivaClient(&http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
		},
	}, conf.Username, conf.Password)
	m.ctx, m.cancel = context.WithCancel(context.Background())
	semaphore := make(chan struct{}, maxGoroutinesPerInterval(conf.MetricConfigs))
	interval := time.Duration(conf.IntervalSeconds) * time.Second
	service := newAccountsService(m.ctx, &m.timeout, m.client)
	utils.RunOnInterval(m.ctx, func() {
		for _, metricConf := range conf.MetricConfigs {
			metricConf.init(service)
			if strings.Contains(metricConf.MetricParameter, "metriclens") {
				m.fetchMetricLensMetrics(interval, semaphore, metricConf)
			} else {
				m.fetchMetrics(interval, semaphore, metricConf)
			}
		}
	}, interval)
	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

func (m *Monitor) fetchMetrics(contextTimeout time.Duration, semaphore chan struct{}, metricConf *metricConfig) {
	select {
	case semaphore <- struct{}{}:
		go func(contextTimeout time.Duration, m *Monitor, url string) {
			defer func() { <-semaphore }()
			ctx, cancel := context.WithTimeout(m.ctx, contextTimeout)
			defer cancel()
			var res map[string]metricResponse
			if _, err := m.client.get(ctx, &res, url); err != nil {
				logger.Errorf("GET metric %s failed. %+v", metricConf.MetricParameter, err)
				return
			}
			var dps []*datapoint.Datapoint
			timestamp := time.Now()
			for metricParameter, series := range res {
				metricConf.logFilterStatuses(series.Meta.FiltersWarmup, series.Meta.FiltersNotExist, series.Meta.FiltersIncompleteData)
				metricName := "conviva." + metricParameter
				for filterID, metricValues := range series.FilterIDValuesMap {
					switch series.Type {
					case "time_series":
						dps = timeSeriesDatapoints(metricName, metricValues, series.Timestamps, metricConf.Account, metricConf.filterName(filterID))
					case "label_series":
						dps = labelSeriesDatapoints(metricName, metricValues, series.Xvalues, timestamp, metricConf.Account, metricConf.filterName(filterID))
					default:
						dps = simpleSeriesDatapoints(metricName, metricValues, timestamp, metricConf.Account, metricConf.filterName(filterID))
					}
					m.sendDatapoints(dps)
				}
			}
		}(contextTimeout, m, fmt.Sprintf(metricURLFormat, metricConf.MetricParameter, metricConf.accountID, strings.Join(metricConf.filterIDs(), ",")))
	}
}

func (m *Monitor) fetchMetricLensMetrics(contextTimeout time.Duration, semaphore chan struct{}, metricConf *metricConfig) {
	for _, dim := range metricConf.MetricLensDimensions {
		select {
		case semaphore <- struct{}{}:
			dim = strings.TrimSpace(dim)
			dimID := metricConf.metricLensDimensionMap[dim]
			if dimID == 0.0 {
				logger.Errorf("No id for MetricLens dimension %s. Wrong MetricLens dimension name.", dim)
				continue
			}
			go func(contextTimeout time.Duration, m *Monitor, metricConf *metricConfig, metricLensDimension string, url string) {
				defer func() { <-semaphore }()
				ctx, cancel := context.WithTimeout(m.ctx, contextTimeout)
				defer cancel()
				var res map[string]metricResponse
				if _, err := m.client.get(ctx, &res, url); err != nil {
					logger.Errorf("GET metric %s failed. %+v", metricConf.MetricParameter, err)
					return
				}
				var dps []*datapoint.Datapoint
				timestamp := time.Now()
				for metricParameter, metricTable := range res {
					metricConf.logFilterStatuses(metricTable.Meta.FiltersWarmup, metricTable.Meta.FiltersNotExist, metricTable.Meta.FiltersIncompleteData)
					for filterID, tableValue := range metricTable.Tables {
						dps = tableDatapoints(metricLensMetrics[metricParameter], metricLensDimension, tableValue.Rows, metricTable.Xvalues, timestamp, metricConf.Account, metricConf.filterName(filterID))
						m.sendDatapoints(dps)
					}
				}
			}(contextTimeout, m, metricConf, dim, fmt.Sprintf(metricLensURLFormat, metricConf.MetricParameter, metricConf.accountID, strings.Join(metricConf.filterIDs(), ","), int(dimID)))
		}
	}
}

func (m *Monitor) sendDatapoints(dps []*datapoint.Datapoint) {
	for i := range dps {
		m.Output.SendDatapoint(dps[i])
	}
}

func maxGoroutinesPerInterval(metricConfigs []*metricConfig) int {
	requests := 0
	for _, metricConfig := range metricConfigs {
		if metricLensDimensionsLength := len(metricConfig.MetricLensDimensions); metricLensDimensionsLength != 0 {
			requests += len(metricConfig.Filters) * metricLensDimensionsLength
		} else {
			requests += len(metricConfig.Filters)
		}
	}
	return int(math.Max(float64(requests), float64(2000)))
}

func timeSeriesDatapoints(metricName string, metricValues []float64, timestamps []int64, accountName string, filterName string) []*datapoint.Datapoint {
	dps := make([]*datapoint.Datapoint, 0)
	metricValue := metricValues[len(metricValues)-1]
	dp := sfxclient.GaugeF(metricName, map[string]string{"account": accountName, "filter": filterName}, metricValue)
	// Series timestamps are in milliseconds
	dp.Timestamp = time.Unix(timestamps[len(timestamps)-1]/1000, 0)
	dp.Meta[dpmeta.NotHostSpecificMeta] = true
	dps = append(dps, dp)
	return dps
}

func labelSeriesDatapoints(metricName string, metricValues []float64, xvalues []string, timestamp time.Time, accountName string, filterName string) []*datapoint.Datapoint {
	dps := make([]*datapoint.Datapoint, 0)
	for i, metricValue := range metricValues {
		dp := sfxclient.GaugeF(metricName, map[string]string{"account": accountName, "filter": filterName}, metricValue)
		dp.Dimensions["label"] = xvalues[i]
		dp.Meta[dpmeta.NotHostSpecificMeta] = true
		dp.Timestamp = timestamp
		dps = append(dps, dp)
	}
	return dps
}

func simpleSeriesDatapoints(metricName string, metricValues []float64, timestamp time.Time, accountName string, filterName string) []*datapoint.Datapoint {
	dps := make([]*datapoint.Datapoint, 0)
	for _, metricValue := range metricValues {
		dp := sfxclient.GaugeF(metricName, map[string]string{"account": accountName, "filter": filterName}, metricValue)
		dp.Meta[dpmeta.NotHostSpecificMeta] = true
		dp.Timestamp = timestamp
		dps = append(dps, dp)
	}
	return dps
}

func tableDatapoints(metricNames []string, dimension string, rows [][]float64, xvalues []string, timestamp time.Time, accountName string, filterName string) []*datapoint.Datapoint {
	dps := make([]*datapoint.Datapoint, 0)
	for rowIndex, row := range rows {
		for metricIndex, metricValue := range row {
			dp := sfxclient.GaugeF(metricNames[metricIndex], map[string]string{"account": accountName, "filter": filterName, dimension: xvalues[rowIndex]}, metricValue)
			dp.Timestamp = timestamp
			dp.Meta[dpmeta.NotHostSpecificMeta] = true
			dps = append(dps, dp)
		}
	}
	return dps
}
