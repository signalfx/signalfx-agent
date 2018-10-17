package conviva

import (
	"context"
	"encoding/json"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	Logger "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const monitorType = "conviva"

var logger = Logger.WithFields(Logger.Fields{"monitorType": monitorType})

// Config for this monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	MetricConfigs []*MetricConfig
	UserName      string    `yaml:"username" validate:"required"`
	Password      string    `yaml:"password" validate:"required"`
	// The maximum amount of time to wait for docker API requests
	TimeoutSeconds int `yaml:"timeoutSeconds" default:"5"`
}

type MetricConfig struct {
	Account           string
	Metric            string `yaml:"metric" default:"quality_metriclens"`
	Filters           []string
	Dimensions        []string
	accountId         string
	filterIds         []string
	filterIdByName    map[string]string
	filterNameById    map[string]string
	dimensionIdByName map[string]string
	dimensionNameById map[string]string
}

// Monitor for conviva metriclens metrics
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

// Configure the monitor and kick off volume metric syncing
func (m *Monitor) Configure(conf *Config) error {
	m.client = &http.Client{}
	if conf.TimeoutSeconds == 0 {
		conf.TimeoutSeconds = 10
	}
	if conf.IntervalSeconds == 0 {
		conf.IntervalSeconds = 5
	}
	m.timeout = time.Duration(conf.TimeoutSeconds) * time.Second
	m.ctx, m.cancel = context.WithCancel(context.Background())
	if conf.MetricConfigs == nil {
		conf.MetricConfigs = []*MetricConfig{&MetricConfig{}}
	}
	configureAccountAndMetric(m.client, conf)
	configureFilters(m.client, conf)
	configureDimensions(m.client, conf)
	utils.RunOnInterval(m.ctx, func() {
		for _, metricConfig := range conf.MetricConfigs {
			m.fetchMetriclensMetrics(metricConfig, conf)
		}
	}, time.Duration(conf.IntervalSeconds) * time.Second)
	//time.Sleep(time.Second * 60)
	return nil
}

// {"default": "c3.NBC", "count": 1, "accounts": {"c3.NBC": "bdcdd5221ca926201cc5d6127dbe5887c25c7f8d", "c3.Demo": "000000000000000000000000000000000000000"}}
func configureAccountAndMetric(client *http.Client, conf *Config)  {
	jsonBody, err := get(client, conf, "https://api.conviva.com/insights/2.4/accounts.json")
	if err != nil {
		logger.Errorf("Get accounts request failed %v\n", err)
		return // nil, err
	}
	accounts := jsonBody["accounts"].(map[string]interface{})
	for _, metricConfig := range conf.MetricConfigs {
		if metricConfig.Metric == "" {
			metricConfig.Metric = "quality_metriclens"
		}
		if metricConfig.Account == "" {
			metricConfig.Account = jsonBody["default"].(string)
			for name, id := range accounts {
				if metricConfig.Account == name {
					metricConfig.accountId = id.(string)
				}
			}
		}
	}
}

func configureFilters(client *http.Client, conf *Config) /*(map[string]map[string]string, error)*/ {
	var jsonBody = make(map[string]map[string]interface{})
	for _, metricConfig := range conf.MetricConfigs {
		if len(jsonBody[metricConfig.accountId]) == 0 {
			var err error
			jsonBody[metricConfig.accountId], err = get(client, conf, "https://api.conviva.com/insights/2.4/filters.json?account=" + metricConfig.accountId)
			if err != nil {
				logger.Errorf("Failed to get filters for account %s: \n%v\n", metricConfig.Account, err)
				continue
			}
		}
		if len(metricConfig.Filters) == 0 {
			metricConfig.Filters = append(metricConfig.Filters, "All Traffic")
		}
		metricConfig.filterIdByName = make(map[string]string)
		metricConfig.filterNameById = make(map[string]string)
		for _, filter := range metricConfig.Filters {
			for filterId, filterName := range jsonBody[metricConfig.accountId] {
				if filter == filterName.(string) {
					metricConfig.filterIds = append(metricConfig.filterIds, filterId)
					metricConfig.filterIdByName[filterName.(string)] = filterId
					metricConfig.filterNameById[filterId] = filterName.(string)
				}
			}
		}
	}
}

func configureDimensions(client *http.Client, conf *Config/*, accounts map[string]idName*/) /*(map[string]map[string]string, error)*/ {
	var jsonBody = make(map[string]map[string]interface{})
	for _, metricConfig := range conf.MetricConfigs {
		if len(jsonBody[metricConfig.accountId]) == 0 {
			var err error
			jsonBody[metricConfig.accountId], err = get(client, conf, "https://api.conviva.com/insights/2.4/metriclens_dimension_list.json?account=" + metricConfig.accountId)
			if err != nil {
				logger.Errorf("Failed to get metriclens dimensions list for account %s: \n%v\n", metricConfig.Account, err)
				continue
			}
		}
		if len(metricConfig.Dimensions) == 0 {
			metricConfig.Dimensions = make([]string, 0, len(jsonBody[metricConfig.accountId]))
			for dimension := range jsonBody[metricConfig.accountId] {
				metricConfig.Dimensions = append(metricConfig.Dimensions, dimension)
			}
		}
		metricConfig.dimensionIdByName = make(map[string]string)
		metricConfig.dimensionNameById = make(map[string]string)
		for _, dimension := range metricConfig.Dimensions {
			if jsonBody[metricConfig.accountId][dimension] != nil {
				dimensionId := strconv.FormatFloat(jsonBody[metricConfig.accountId][dimension].(float64), 'f', 0, 64)
				metricConfig.dimensionIdByName[dimension]   = dimensionId
				metricConfig.dimensionNameById[dimensionId] = dimension
			}
		}
	}
}

func (m *Monitor) fetchMetriclensMetrics(metricConfig *MetricConfig, conf *Config) {
	_, cancel := context.WithTimeout(m.ctx, m.timeout)
	defer cancel()
	for _, dimension := range metricConfig.Dimensions {
		url := "https://api.conviva.com/insights/2.4/metrics.json?metrics=" + metricConfig.Metric + "&filter_ids=" + strings.Join(metricConfig.filterIds, ",") + "&metriclens_dimension_id=" + metricConfig.dimensionIdByName[dimension]
		go m.fetchMetriclensMetricsWithContext(conf, url, metricConfig, dimension)
		//go test(url)
	}
	//select {
	//case <-ctx.Done():
	//	fmt.Println(ctx.Err())
	//}
}

//func test(url string)  {
//	time.Sleep(time.Second * 15)
//	fmt.Println("hello")
//
//}

func (m *Monitor) fetchMetriclensMetricsWithContext(conf *Config, url string, metricConfig *MetricConfig, dimension string) {
	time.Sleep(time.Second * 15)
	tableTypeResponse, err := get(m.client, conf, url)
	if err != nil {
		logger.WithError(err).Error("Could not get conviva metrics")
		return
	}
	tableTypeResponse["account"] = metricConfig.Account
	tableTypeResponse["accountId"] = metricConfig.accountId
	tableTypeResponse["dimension"] = dimension
	tableTypeResponse["dimensionId"] = metricConfig.dimensionIdByName[dimension]
	tableTypeResponse["filterNameById"] = metricConfig.filterNameById
	datapoints, err := jsonResponseToDatapoints(tableTypeResponse)
	if err != nil {
		logger.WithError(err).Error("Could not convert conviva metrics to datapoints")
		return
	}
	now := time.Now()
	for i := range datapoints {
		datapoints[i].Timestamp = now
		m.Output.SendDatapoint(datapoints[i])
	}

}

//{"quality_metriclens": {"tables": {"19457": {"rows": [], "total_row": [1444, 0.8310249307479225, 10.457063711911358, 88.78116343490305, 3.37, 0.63, 3453.2393199999997, 0.583941605839416, 1370, 0.46, 6.47]}}, "meta": {"status": 0, "filters_warmup": [19457]}, "type": "table", "xvalues": []}}

func get(client *http.Client, conf *Config, url string) (map[string]interface{}, error) {
	request, _ := http.NewRequest("GET", url, nil)
	request.SetBasicAuth(conf.UserName, conf.Password)
	//if err != nil {
	//	log.Fatal(err)
	//}
	response, err := client.Do(request)

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		logger.WithError(err).Error("Could not get prometheus metrics")
		logger.Fatal(err)
	}
	var payload map[string]interface{}
	err = json.Unmarshal(body, &payload)
	return payload, err
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

