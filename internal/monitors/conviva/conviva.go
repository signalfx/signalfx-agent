package conviva

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/signalfx-agent/internal/core/common/dpmeta"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
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

// Config for this monitor
type Config struct {
	config.MonitorConfig
	// Conviva Pulse username required with each API request.
	Username                    string                                 `yaml:"pulseUsername" validate:"required"`
	// Conviva Pulse password required with each API request.
	Password                              string                       `yaml:"pulsePassword" validate:"required"`
	TimeoutSeconds                        int                          `yaml:"timeoutSeconds" default:"15"`
	MetricConfigs                         []*MetricConfig              `yaml:"metricConfigs"`
	defaultAccount                        string
	accounts                              map[string]string
	filterByAccountAndID                  map[string]map[string]string
	filterIDByAccountAndName              map[string]map[string]string
	metriclensDimensionIDByAccountAndName map[string]map[string]string
	metricLensFilterIDByAccountAndName    map[string]map[string]string

}

// MetricConfig for configuring individual metric
type MetricConfig struct {
	// Conviva customer account name. The default account is used if not specified.
	Account              string   `yaml:"account"`
	Metric               string   `yaml:"metric" default:"quality_metriclens"`
	// Filter names. The default is `All Traffic` filter
	Filters              []string `yaml:"filters"`
	// Metriclens dimension names.
	MetriclensDimensions []string `yaml:"metriclensDimensions"`
	accountID            string
	filterIDs            []string
}

// Monitor for conviva metrics
type Monitor struct {
	Output  types.Output
	cancel  context.CancelFunc
	ctx     context.Context
	client  *http.Client
	timeout time.Duration
}

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Configure monitor
func (m *Monitor) Configure(conf *Config) error {
	m.timeout = time.Duration(conf.TimeoutSeconds) * time.Second
	m.client = &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
		},
	}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	initConfig(m, conf)
	semaphore := make(chan struct{}, maxGoroutinesPerInterval(conf.MetricConfigs))
	utils.RunOnInterval(m.ctx, func() {
		initConfig(m, conf)
		for _, metricConfig := range conf.MetricConfigs {
			if strings.Contains(metricConfig.Metric, "metriclens") {
				fetchMetriclensMetrics(semaphore, m, metricConfig, conf)
			} else {
				fetchMetrics(semaphore, m, metricConfig, conf)
			}
		}
	}, time.Duration(conf.IntervalSeconds) * time.Second)
	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

func fetchMetrics(semaphore chan struct{}, m *Monitor, metricConfig *MetricConfig, conf *Config) {
	select {
	case semaphore <- struct{}{}:
		go func(m *Monitor, conf *Config, url string) {
			defer func() { <- semaphore }()
			ctx, cancel := context.WithTimeout(m.ctx, time.Duration(conf.IntervalSeconds) * time.Second)
			defer cancel()
			res, err := get(ctx, m, conf, url)
			if err != nil {
				logger.Error(err)
				return
			}
			dps := make([]*datapoint.Datapoint, 0)
			metric := metricConfig.Metric
			account := metricConfig.Account
			if res[metric] != nil && res[metric].(map[string]interface{}) != nil && res[metric].(map[string]interface{})["filters"] != nil && res[metric].(map[string]interface{})["type"] != nil {
				// series can be of type time series, simple series or label series
				for filterID, series := range res[metric].(map[string]interface{})["filters"].(map[string]interface{}) {
					if series == nil { continue }
					for i, metricValue := range series.([]interface{}) {
						select {
						case <- ctx.Done():
							logger.Error(ctx.Err())
							return
						default:
							if metricValue == nil { continue }
							dp := sfxclient.GaugeF(
								metric,
								map[string]string{
									"account": account,
									"filter":  conf.filterByAccountAndID[account][filterID],
								},
								metricValue.(float64))
							// Checking the type of series and setting dimensions accordingly
							switch res[metric].(map[string]interface{})["type"].(string) {
							case "time_series":
								timestamps := res[metric].(map[string]interface{})["timestamps"]
								if timestamps == nil || timestamps.([]interface{}) == nil || timestamps.([]interface{})[i] == nil {
									continue
								}
								dp.Timestamp = time.Unix(int64(0.001 * timestamps.([]interface{})[i].(float64)), 0)
							case "label_series":
								xvalues := res[metric].(map[string]interface{})["xvalues"]
								if xvalues == nil || xvalues.([]interface{}) == nil || xvalues.([]interface{})[i] == nil {
									continue
								}
								dp.Dimensions["label"] = xvalues.([]interface{})[i].(string)
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
		}(m, conf, "https://api.conviva.com/insights/2.4/metrics.json?metrics="+metricConfig.Metric+"&filter_ids="+strings.Join(metricConfig.filterIDs, ","))
	}
}

func fetchMetriclensMetrics(semaphore chan struct{}, m *Monitor, metricConfig *MetricConfig, conf *Config)  {
	for _, metriclensDimension := range metricConfig.MetriclensDimensions {
		select {
		case semaphore <- struct{}{}:
			metriclensDimension = strings.TrimSpace(metriclensDimension)
			metriclensDimensionID := conf.metriclensDimensionIDByAccountAndName[metricConfig.Account][metriclensDimension]
			if metriclensDimensionID == "" {
				logger.Errorf("No id for metriclens dimension %s. Wrong metriclens dimension name.", metriclensDimension)
				continue
			}
			go func(m *Monitor, conf *Config, metricConfig *MetricConfig, metriclensDimension string, url string) {
				defer func() { <- semaphore }()
				ctx, cancel := context.WithTimeout(m.ctx, time.Duration(conf.IntervalSeconds) * time.Second)
				defer cancel()
				res, err := get(ctx, m, conf, url)
				if err != nil {
					logger.Error(err)
					return
				}
				dps := make([]*datapoint.Datapoint, 0)
				account := metricConfig.Account
				metric := metricConfig.Metric
				if res[metric] != nil && res[metric].(map[string]interface{})["tables"] != nil {
					for filterID, table := range res[metric].(map[string]interface{})["tables"].(map[string]interface{}) {
						if table == nil { continue }
						for tableKey, tableValue := range table.(map[string]interface{}) {
							if tableKey == "rows" {
								if tableValue == nil { continue }
								for rowIndex, row := range tableValue.([]interface{}) {
									select {
									case <- ctx.Done():
										logger.Error(ctx.Err())
										return
									default:
										if row == nil { continue }
										xvalues := res[metric].(map[string]interface{})["xvalues"]
										for metricIndex, metricValue := range row.([]interface{}) {
											if metricValue == nil || xvalues == nil || xvalues.([]interface{})[rowIndex] == nil {
												continue
											}
											dps = append(dps, sfxclient.GaugeF(
												metric,
												map[string]string{
													"account":           account,
													"filter":            conf.filterByAccountAndID[account][filterID],
													"metriclensMetric":  metriclensMetrics[metric][metricIndex],
													metriclensDimension: xvalues.([]interface{})[rowIndex].(string),
												},
												metricValue.(float64)))
										}
									}
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
			}(m, conf, metricConfig, metriclensDimension, "https://api.conviva.com/insights/2.4/metrics.json?metrics="+metricConfig.Metric+"&filter_ids="+strings.Join(metricConfig.filterIDs, ",")+"&metriclens_dimension_id="+metriclensDimensionID)
		}
	}
}

func get(ctx context.Context, m *Monitor, conf *Config, url string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	req.SetBasicAuth(conf.Username, conf.Password)
	res, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var payload map[string]interface{}
	err = json.Unmarshal(body, &payload)
	if res.StatusCode != 200 && payload != nil {
		return nil, fmt.Errorf("response status code: %d. Reason: %s", res.StatusCode, payload["reason"])
	}
	return payload, err
}

func maxGoroutinesPerInterval(metricConfigs []*MetricConfig) int {
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
