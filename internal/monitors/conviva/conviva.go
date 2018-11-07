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

// MONITOR(Conviva): This monitor uses version 2.4 of the Conviva Experience Insights REST APIs to pull
// `Real-Time/Live` video playing experience metrics from Conviva.
//
// Only `Live` conviva metrics listed
// [here](https://community.conviva.com/site/global/apis_data/experience_insights_api/index.gsp#metrics)
// are supported. The metrics are gauges. They are converted to SignalFx metrics with dimensions for the
// account name, filter name. In the case of metriclenses, the names of the constituent metrics and the
// conviva metriclens dimensions are included. The values of the conviva dimensions are derived from
// the values of the associated metriclens dimension entities.
//
// Below is a sample YAML configuration showing the most basic configuration of the conviva monitor
// using the required fields. For this configuration the monitor will default to fetching quality metriclens
// metrics from the default conviva account using the `All Traffic` filter.
//
// ```
//monitors:
//- type: conviva
//  pulseUsername: <username>
//  pulsePassword: <password>
// ```
//
// Individual metrics are configured in a list of metricConfigs as shown in sample configuration below.
// Metric values are the titles of the metrics
// [here](https://github.com/signalfx/integrations/tree/master/conviva/docs) which are the same as
// the Conviva metric parameters
// [here](https://community.conviva.com/site/global/apis_data/experience_insights_api/index.gsp#metrics)
// Where an account is not provided the default account is used. Where no filters are specified the
// `All Traffic` filter is used. Where metriclens dimensions are not specified all metriclens dimensions
// are used. The `_ALL_` keyword means all. Dimensions only apply to metriclenses. If specified for a
// regular metric they will be ignored.
//
// ```
//monitors:
//- type: conviva
//  pulseUsername: <username>
//  pulsePassword: <password>
//  metricConfigs:
//    - account: c3.NBC
//      metric: audience_metriclens
//      filters:
//        - All Traffic
//      metriclensDimensions:
//        - Cities
//    - metric: avg_bitrate
//      filters:
//        - _ALL_
//    - metric: concurrent_plays
//    - metric: quality_metriclens
//      filters:
//        - All Traffic
//      metriclensDimensions:
//        - _ALL_
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
	Username       string          `yaml:"pulseUsername" validate:"required"`
	// Conviva Pulse password required with each API request.
	Password       string          `yaml:"pulsePassword" validate:"required" neverLog:"true"`
	TimeoutSeconds int             `yaml:"timeoutSeconds" default:"10"`
	MetricConfigs  []*metricConfig `yaml:"metricConfigs"`
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
		conf.MetricConfigs = []*metricConfig{{Metric: "quality_metriclens"}}
	}
	m.timeout = time.Duration(conf.TimeoutSeconds) * time.Second
	m.client = newConvivaClient(&http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
		},
	}, conf.Username , conf.Password)

	m.ctx, m.cancel = context.WithCancel(context.Background())
	semaphore := make(chan struct{}, maxGoroutinesPerInterval(conf.MetricConfigs))
	interval := time.Duration(conf.IntervalSeconds) * time.Second
	service := newAccountsService(m.ctx, &m.timeout, &m.client)
	utils.RunOnInterval(m.ctx, func() {
		for _, metricConf := range conf.MetricConfigs {
			metricConf.init(service)
			if strings.Contains(metricConf.Metric, "metriclens") {
				m.fetchMetriclensMetrics(interval, semaphore, metricConf)
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

func (m *Monitor) fetchMetrics(contextTimeout time.Duration, semaphore chan struct{}, conf *metricConfig) {
	select {
	case semaphore <- struct{}{}:
		go func(contextTimeout time.Duration, m *Monitor, url string) {
			defer func() { <- semaphore }()
			ctx, cancel := context.WithTimeout(m.ctx, contextTimeout)
			defer cancel()
			var res map[string]metricResponse
			if _, err := m.client.get(ctx, &res, url); err != nil {
				logger.Errorf("GET metric %s failed. %+v", conf.Metric, err)
				return
			}
			dps := make([]*datapoint.Datapoint, 0)
			for metricName, series := range res {
				series.Meta.logErrorFilterStatus(url)
				prefixedMetricName := "conviva." + metricName
				for filterID, metricValues := range series.FilterIDValuesMap {
					for i, metricValue := range metricValues {
						select {
						case <-ctx.Done(): logger.Error(ctx.Err()); return
						default:
							dp := sfxclient.GaugeF(
								prefixedMetricName,
								map[string]string{"account": conf.Account, "filter": conf.filterMap[filterID],},
								metricValue)
							// Checking the type of series and setting dimensions accordingly
							switch series.Type {
							case "time_series":
								dp.Timestamp = time.Unix(int64(0.001 * series.Timestamps[i]), 0)
							case "label_series":
								dp.Dimensions["label"] = series.Xvalues[i]
								fallthrough
							default:
								dp.Timestamp = time.Now()
							}
							dp.Meta[dpmeta.NotHostSpecificMeta] = true
							dps = append(dps, dp)
						}
					}
				}
			}
			for i := range dps {
				m.Output.SendDatapoint(dps[i])
			}
		}(contextTimeout, m, fmt.Sprintf(metricURLFormat, conf.Metric, conf.accountID, strings.Join(conf.filterIDs(), ",")))
	}
}

func (m *Monitor) fetchMetriclensMetrics(contextTimeout time.Duration, semaphore chan struct{}, conf *metricConfig)  {
	for _, dim := range conf.MetriclensDimensions {
		select {
		case semaphore <- struct{}{}:
			dim = strings.TrimSpace(dim)
			dimID := conf.metriclensDimensionMap[dim]
			if dimID == 0.0 {
				logger.Errorf("No id for metriclens dimension %s. Wrong metriclens dimension name.", dim)
				continue
			}
			go func(contextTimeout time.Duration, m *Monitor, conf *metricConfig, metriclensDimension string, url string) {
				defer func() { <- semaphore }()
				ctx, cancel := context.WithTimeout(m.ctx, contextTimeout)
				defer cancel()
				var res map[string]metricResponse
				if _, err := m.client.get(ctx, &res, url); err != nil {
					logger.Errorf("GET metric %s failed. %+v", conf.Metric, err)
					return
				}
				dps := make([]*datapoint.Datapoint, 0)
				for metricName, metricTable := range res {
					metricTable.Meta.logErrorFilterStatus(url)
					for filterID, tableValue := range metricTable.Tables {
						for rowIndex, row := range tableValue.Rows {
							select {
							case <- ctx.Done():
								logger.Error(ctx.Err())
								return
							default:
								for metricIndex, metricValue := range row {
									dps = append(dps, sfxclient.GaugeF(
										prefixedMetriclensMetrics[metricName][metricIndex],
										map[string]string{
											"account":           conf.Account,
											"filter":            conf.filterMap[filterID],
											metriclensDimension: metricTable.Xvalues[rowIndex],
										},
										metricValue))
								}
							}
						}
					}
				}
				now := time.Now()
				for i := range dps {
					dps[i].Timestamp = now
					dps[i].Meta[dpmeta.NotHostSpecificMeta] = true
					m.Output.SendDatapoint(dps[i])
				}
			}(contextTimeout, m, conf, dim, fmt.Sprintf(metricLensURLFormat, conf.Metric, conf.accountID, strings.Join(conf.filterIDs(), ","), int(dimID)))
		}
	}
}

func maxGoroutinesPerInterval(metricConfigs []*metricConfig) int {
	requests := 0
	for _, metricConfig := range metricConfigs {
		if metriclensDimensionsLength := len(metricConfig.MetriclensDimensions); metriclensDimensionsLength != 0 {
			requests += len(metricConfig.Filters) * metriclensDimensionsLength
		} else {
			requests += len(metricConfig.Filters)
		}
	}
	return int(math.Max(float64(requests), float64(2000)))
}
