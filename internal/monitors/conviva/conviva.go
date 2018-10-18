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
	config.MonitorConfig //`yaml:",inline" acceptsEndpoints:"true"`
	MetricConfigs []*MetricConfig `yaml:"metricConfigs"`
	UserName      string          `yaml:"pulse_username" validate:"required"`
	Password      string          `yaml:"pulse_password" validate:"required"`
	// The maximum amount of time to wait for docker API requests
	TimeoutSeconds int `yaml:"timeoutSeconds" default:"5"`
}

type MetricConfig struct {
	Account           string   `yaml:"account"`
	Metric            string   `yaml:"metric" default:"quality_metriclens"`
	Filters           []string `yaml:"filters"`
	Dimensions        []string `yaml:"dimensions"`
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

//func main(){
//	m := Monitor{}
//	m.Configure(&Config{config.MonitorConfig{IntervalSeconds: 10}, nil, os.Getenv("username"), os.Getenv("password"),  10})
//}

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Configure the monitor and kick off volume metric syncing
func (m *Monitor) Configure(conf *Config) error {
	m.client = &http.Client{}
	logger.Debugf("configuration object before additional auto-configuration:\n%+v\n", conf)
	m.timeout = time.Duration(conf.TimeoutSeconds) * time.Second
	m.ctx, m.cancel = context.WithCancel(context.Background())
	if conf.MetricConfigs == nil {
		conf.MetricConfigs = []*MetricConfig{{Metric: "quality_metriclens"}}
	}
	configureAccount(m.client, conf)
	configureFilters(m.client, conf)
	configureDimensions(m.client, conf)
	logger.Debugf("configuration object after additional auto-configuration:\n%+v\n", conf)
	utils.RunOnInterval(m.ctx, func() {
		for _, metricConfig := range conf.MetricConfigs {
			fetchMetriclensMetrics(m, metricConfig, conf)
		}
	}, time.Duration(conf.IntervalSeconds) * time.Second)
	//time.Sleep(time.Second * 60)
	return nil
}

func fetchMetriclensMetrics(m *Monitor, metricConfig *MetricConfig, conf *Config) {
	_, cancel := context.WithTimeout(m.ctx, m.timeout)
	defer cancel()
	for _, dimension := range metricConfig.Dimensions {
		url := "https://api.conviva.com/insights/2.4/metrics.json?metrics=" + metricConfig.Metric + "&filter_ids=" + strings.Join(metricConfig.filterIds, ",") + "&metriclens_dimension_id=" + metricConfig.dimensionIdByName[dimension]
		go func(m *Monitor, conf *Config, url string, metricConfig *MetricConfig, dimension string) {
			//time.Sleep(time.Second * 10)
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
		}(m, conf, url, metricConfig, dimension)
	}
	//select {
	//case <-ctx.Done():
	//	fmt.Println(ctx.Err())
	//}
}

// {"default": "c3.NBC", "count": 1, "accounts": {"c3.NBC": "bdcdd5221ca926201cc5d6127dbe5887c25c7f8d", "c3.Demo": "000000000000000000000000000000000000000"}}
func configureAccount(client *http.Client, conf *Config)  {
	jsonResponse, err := get(client, conf, "https://api.conviva.com/insights/2.4/accounts.json")
	if err != nil {
		logger.Errorf("Get accounts request failed %v\n", err)
		return // nil, err
	}
	accounts := jsonResponse["accounts"].(map[string]interface{})
	for _, metricConfig := range conf.MetricConfigs {
		if metricConfig.Account == "" {
			metricConfig.Account = jsonResponse["default"].(string)
			for name, id := range accounts {
				if metricConfig.Account == name {
					metricConfig.accountId = id.(string)
				}
			}
		}
	}
}

func configureFilters(client *http.Client, conf *Config) {
	var jsonResponse = make(map[string]map[string]interface{})
	for _, metricConfig := range conf.MetricConfigs {
		if len(jsonResponse[metricConfig.accountId]) == 0 {
			var err error
			jsonResponse[metricConfig.accountId], err = get(client, conf, "https://api.conviva.com/insights/2.4/filters.json?account=" + metricConfig.accountId)
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
			for filterId, filterName := range jsonResponse[metricConfig.accountId] {
				if filter == filterName.(string) {
					metricConfig.filterIds = append(metricConfig.filterIds, filterId)
					metricConfig.filterIdByName[filterName.(string)] = filterId
					metricConfig.filterNameById[filterId] = filterName.(string)
				}
			}
		}
	}
}

func configureDimensions(client *http.Client, conf *Config) {
	var jsonResponse = make(map[string]map[string]interface{})
	for _, metricConfig := range conf.MetricConfigs {
		if len(jsonResponse[metricConfig.accountId]) == 0 {
			var err error
			jsonResponse[metricConfig.accountId], err = get(client, conf, "https://api.conviva.com/insights/2.4/metriclens_dimension_list.json?account=" + metricConfig.accountId)
			if err != nil {
				logger.Errorf("Failed to get metriclens dimensions list for account %s: \n%v\n", metricConfig.Account, err)
				continue
			}
		}
		if len(metricConfig.Dimensions) == 0 {
			metricConfig.Dimensions = make([]string, 0, len(jsonResponse[metricConfig.accountId]))
			for dimension := range jsonResponse[metricConfig.accountId] {
				metricConfig.Dimensions = append(metricConfig.Dimensions, dimension)
			}
		}
		metricConfig.dimensionIdByName = make(map[string]string)
		metricConfig.dimensionNameById = make(map[string]string)
		for _, dimension := range metricConfig.Dimensions {
			if jsonResponse[metricConfig.accountId][dimension] != nil {
				dimensionId := strconv.FormatFloat(jsonResponse[metricConfig.accountId][dimension].(float64), 'f', 0, 64)
				metricConfig.dimensionIdByName[dimension]   = dimensionId
				metricConfig.dimensionNameById[dimensionId] = dimension
			}
		}
	}
}

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

