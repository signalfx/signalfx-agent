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
	"net/http"
	"strings"
	"time"
)

const monitorType = "conviva"

// MONITOR(conviva): This monitor uses version 2.4 of the Conviva Experience Insights REST APIs to pull
// `real-time/live` video playing experience metrics from Conviva.
//
// Only `real-time/live` conviva metrics listed
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
// `All Traffic` filter is used. Where dimensions are not specified all dimensions are used. The `*`
// wildcard means all. Dimensions only apply to metriclenses. If specified for a regular metric they
// will be ignored.
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
//      dimensions:
//        - Cities
//    - metric: avg_bitrate
//      filters:
//        - *
//    - metric: concurrent_plays
//    - metric: quality_metriclens
//      filters:
//        - All Traffic
//      dimensions:
//        - *
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
	Username                    string                       `yaml:"pulseUsername" validate:"required"`
	// Conviva Pulse password required with each API request.
	Password                    string                       `yaml:"pulsePassword" validate:"required"`
	TimeoutSeconds              int                          `yaml:"timeoutSeconds" default:"15"`
	MetricConfigs               []*MetricConfig              `yaml:"metricConfigs"`
	filterNameByID              map[string]map[string]string
	dimensionIDByAccountAndName map[string]map[string]string
	metriclensMetricNames       map[string][]string
}

// MetricConfig for configuring individual metric
type MetricConfig struct {
	// Conviva customer account name. The default account is used if not specified.
	Account             string   `yaml:"account"`
	Metric              string   `yaml:"metric" default:"quality_metriclens"`
	// Filter names. The default is `All Traffic` filter
	Filters             []string `yaml:"filters"`
	// Metriclens dimension names.
	Dimensions          []string `yaml:"dimensions"`
	accountID           string
	filterIDs           []string
	metriclensFilterIDs []string
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
	logger.Debugf("configuration object before additional auto-configuration:\n%+v\n", conf)
	m.timeout = time.Duration(conf.TimeoutSeconds) * time.Second
	m.client = &http.Client{
		Timeout: m.timeout,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
		},
	}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	setConfigFields(m, conf)
	maxNumOfGoroutines := len(conf.MetricConfigs) * 100
	if maxNumOfGoroutines > 500 {
		maxNumOfGoroutines = 500
	}
	maxNumOfGoroutinesChan := make(chan struct{}, maxNumOfGoroutines)
	utils.RunOnInterval(m.ctx, func() {
		for _, metricConfig := range conf.MetricConfigs {
			if strings.Contains(metricConfig.Metric, "metriclens") {
				getSendMetriclensMetrics(maxNumOfGoroutinesChan, m, metricConfig, conf)
			} else {
				getSendMetrics(maxNumOfGoroutinesChan, m, metricConfig, conf)
			}
		}
	}, time.Duration(conf.IntervalSeconds) * time.Second)
	return nil
}

func getSendMetrics(maxNumOfGoroutinesChan chan struct{}, m *Monitor, metricConfig *MetricConfig, conf *Config) {
	ctx, _ := context.WithTimeout(m.ctx, time.Duration(conf.IntervalSeconds) * time.Second)
	select {
	case <- ctx.Done():
		logger.Error(ctx.Err())
		return
	case maxNumOfGoroutinesChan <- struct{}{}:
		url := "https://api.conviva.com/insights/2.4/metrics.json?metrics=" + metricConfig.Metric +
			"&filter_ids=" + strings.Join(metricConfig.filterIDs, ",")
		go func(m *Monitor, conf *Config, url string) {
			defer func() {<- maxNumOfGoroutinesChan}()
			res, err := get(m, conf, url)
			if err != nil {
				logger.Error(err)
				return
			}
			// The "series" in seriesByFilterID could be of type time series, simple series or label series
			seriesByFilterID := res[metricConfig.Metric].(map[string]interface{})["filters"].(map[string]interface{})
			dps := make([]*datapoint.Datapoint, 0)
			for filterID, series := range seriesByFilterID {
				for i, metricValue := range series.([]interface{}) {
					dp := sfxclient.GaugeF(
						metricConfig.Metric,
						map[string]string{
							//TODO: Redundant dimension. Get rid of it
							"metric":  metricConfig.Metric,
							"account": metricConfig.Account,
							"filter":  conf.filterNameByID[metricConfig.Account][filterID],
						},
						metricValue.(float64))
					// Checking the type of series and setting dimensions accordingly
					switch res[metricConfig.Metric].(map[string]interface{})["type"].(string) {
					case "time_series":
						dp.Timestamp = time.Unix(int64(0.001 * res[metricConfig.Metric].(map[string]interface{})["timestamps"].([]interface{})[i].(float64)), 0)
					case "label_series":
						dp.Dimensions["label"] = res[metricConfig.Metric].(map[string]interface{})["xvalues"].([]interface{})[i].(string)
						fallthrough
					default:
						dp.Timestamp = time.Now()
					}
					dp.Meta[dpmeta.NotHostSpecificMeta] = true
					dps = append(dps, dp)
				}
			}
			for i := range dps {

				m.Output.SendDatapoint(dps[i])
			}
		}(m, conf, url)
	}
}

func getSendMetriclensMetrics(maxNumOfGoroutinesChan chan struct{}, m *Monitor, metricConfig *MetricConfig, conf *Config)  {
	ctx, _ := context.WithTimeout(m.ctx, time.Duration(conf.IntervalSeconds) * time.Second)
	for _, metriclensDimension := range metricConfig.Dimensions {
		select {
		case <- ctx.Done():
			logger.Error(ctx.Err())
			return
		case maxNumOfGoroutinesChan <- struct{}{}:
			url := "https://api.conviva.com/insights/2.4/metrics.json?metrics=" + metricConfig.Metric +
				"&filter_ids=" + strings.Join(metricConfig.metriclensFilterIDs, ",") +
				"&metriclens_dimension_id=" + conf.dimensionIDByAccountAndName[metricConfig.Account][metriclensDimension]
			go func(m *Monitor, conf *Config, url string, metricConfig *MetricConfig, metriclensDimension string) {
				defer func() {<- maxNumOfGoroutinesChan}()
				res, err := get(m, conf, url)
				if err != nil {
					logger.Error(err)
					return
				}
				tablesByFilterID := res[metricConfig.Metric].(map[string]interface{})["tables"].(map[string]interface{})
				metriclensDimensionEntities := res[metricConfig.Metric].(map[string]interface{})["xvalues"].([]interface{})
				dps := make([]*datapoint.Datapoint, 0)
				for filterID, table := range tablesByFilterID {
					for tableKey, tableValue := range table.(map[string]interface{}) {
						if tableKey == "rows" {
							for rowIndex, row := range tableValue.([]interface{}) {
								select {
								case <-ctx.Done():
									logger.Error(ctx.Err())
									return
								default:
									if row != nil {
										for metricIndex, metricValue := range row.([]interface{}) {
											dps = append(dps, sfxclient.GaugeF(
												conf.metriclensMetricNames[metricConfig.Metric][metricIndex],
												map[string]string{
													"metric":  metricConfig.Metric,
													"account": metricConfig.Account,
													"filter":  conf.filterNameByID[metricConfig.Account][filterID],
													metriclensDimension: metriclensDimensionEntities[rowIndex].(string),
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
			}(m, conf, url, metricConfig, metriclensDimension)
		}
	}
}

func get(m *Monitor, conf *Config, url string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
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
		return nil, fmt.Errorf("Response status code: %d. Reason: %s", res.StatusCode, payload["reason"])
	}
	return payload, err
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}