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
	Logger "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const monitorType = "conviva"

// MONITOR(conviva): This monitor uses version 2.4 of the Conviva Experience Insights REST APIs to pull
// `real-time(live)` video playing experience metrics from Conviva.
//
// Only `live` metrics types are supported. See
// https://community.conviva.com/site/global/apis_data/experience_insights_api/index.gsp#metrics
// if Conviva developer community member. The live metrics are gauge type metrics. They are converted
// to SignalFx gauges with dimensions for the metric name, account name and filter name. For metriclens,
// the metriclens dimensions are converted to SignalFx dimensions in addition. The values for these
// dimensions are derived from the values of the associated metriclens dimension entities.
//
// TODO: doc about the default behavior
//
// Sample YAML configuration:
//
// ```
//monitors:
//- type: conviva
//  pulse_username: <username>
//  pulse_password: <password>
//  timeoutSeconds: 20
//  intervalSeconds: 25
//  metricConfigs:
//    - account: c3.NBC
//      metric: quality_metriclens
//      filters:
//        - All Traffic
//        - Live
//      dimensions:
//        - Cities
//        - CDNs
//    - account: c3.NBC
//      metric: avg_bitrate
//      filters:
//        - All Traffic
// ```

var logger = Logger.WithFields(Logger.Fields{"monitorType": monitorType})

// Config for this monitor
type Config struct {
	config.MonitorConfig
	Username                    string                       `yaml:"pulse_username" validate:"required"`
	Password                    string                       `yaml:"pulse_password" validate:"required"`
	TimeoutSeconds              int                          `yaml:"timeoutSeconds" default:"5"`
	MetricConfigs               []*MetricConfig              `yaml:"metricConfigs"`
	filterNameById              map[string]map[string]string
	dimensionIdByAccountAndName map[string]map[string]string
	metriclensMetricNames       map[string][]string
}

type MetricConfig struct {
	Account             string   `yaml:"account"`
	Metric              string   `yaml:"metric" default:"quality_metriclens"`
	Filters             []string `yaml:"filters"`
	Dimensions          []string `yaml:"dimensions"`
	accountId           string
	filterIds           []string
	metriclensFilterIds []string
}

// Monitor for conviva metrics
type Monitor struct {
	Output types.Output
	cancel func()
	ctx     context.Context
	client *http.Client
	timeout time.Duration
}

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

func (m *Monitor) Configure(conf *Config) error {
	logger.Debugf("configuration object before additional auto-configuration:\n%+v\n", conf)
	m.timeout = time.Duration(conf.TimeoutSeconds) * time.Second
	m.client = &http.Client{
		Timeout: m.timeout,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
		},
	}
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
//TODO: implement functionality for simple series and label series
func getSendMetrics(maxNumOfGoroutinesChan chan struct{}, m *Monitor, metricConfig *MetricConfig, conf *Config) {
	ctx, cancel := context.WithTimeout(m.ctx, time.Duration(conf.IntervalSeconds) * time.Second)
	select {
	case <- ctx.Done():
		cancel()
		logger.Error(ctx.Err())
		return
	case maxNumOfGoroutinesChan <- struct{}{}:
		url := "https://api.conviva.com/insights/2.4/metrics.json?metrics=" + metricConfig.Metric +
			"&filter_ids=" + strings.Join(metricConfig.filterIds, ",")
		go func(m *Monitor, conf *Config, url string) {
			defer func() {<- maxNumOfGoroutinesChan}()
			res, err := get(m, conf, url)
			if err != nil {
				logger.Error(err)
				return
			}
			timestamps := res[metricConfig.Metric].(map[string]interface{})["timestamps"].([]interface{})
			timeSeriesByFilterId := res[metricConfig.Metric].(map[string]interface{})["filters"].(map[string]interface{})
			dps := make([]*datapoint.Datapoint, 0)
			for filterId, timeSeries := range timeSeriesByFilterId {
				for i, metricValue := range timeSeries.([]interface{}) {
					dp := sfxclient.GaugeF(
						metricConfig.Metric,
						map[string]string{
							"metric":  metricConfig.Metric,
							"account": metricConfig.Account,
							"filter":  conf.filterNameById[metricConfig.Account][filterId],
						},
						metricValue.(float64))
					dp.Timestamp = time.Unix(int64(0.001 * timestamps[i].(float64)), 0)
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
	ctx, cancel := context.WithTimeout(m.ctx, time.Duration(conf.IntervalSeconds) * time.Second)
	for _, metriclensDimension := range metricConfig.Dimensions {
		select {
		case <- ctx.Done():
			cancel()
			logger.Error(ctx.Err())
			return
		case maxNumOfGoroutinesChan <- struct{}{}:
			url := "https://api.conviva.com/insights/2.4/metrics.json?metrics=" + metricConfig.Metric +
				"&filter_ids=" + strings.Join(metricConfig.metriclensFilterIds, ",") +
				"&metriclens_dimension_id=" + conf.dimensionIdByAccountAndName[metricConfig.Account][metriclensDimension]
			go func(m *Monitor, conf *Config, url string, metricConfig *MetricConfig, metriclensDimension string) {
				defer func() {<- maxNumOfGoroutinesChan}()
				res, err := get(m, conf, url)
				if err != nil {
					logger.Error(err)
					return
				}
				tablesByFilterId := res[metricConfig.Metric].(map[string]interface{})["tables"].(map[string]interface{})
				metriclensDimensionEntities := res[metricConfig.Metric].(map[string]interface{})["xvalues"].([]interface{})
				dps := make([]*datapoint.Datapoint, 0)
				for filterId, table := range tablesByFilterId {
					for tableKey, tableValue := range table.(map[string]interface{}) {
						if tableKey == "rows" {
							for rowIndex, row := range tableValue.([]interface{}) {
								select {
								case <-ctx.Done():
									cancel()
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
													"filter":  conf.filterNameById[metricConfig.Account][filterId],
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
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Response status code: %d. Reason: %s.", res.StatusCode, payload["reason"])
	}
	return payload, err
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}