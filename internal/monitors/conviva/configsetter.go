package conviva

import (
	"strconv"
	"strings"
)

func setConfigFields(m *Monitor, conf *Config)  {
	conf.metriclensMetricNames = map[string][]string{
		"quality_metriclens": {
			"total_attempts",
			"video_start_failures_percent",
			"exits_before_video_start_percent",
			"plays_percent",
			"video_startup_time_sec",
			"rebuffering_ratio_percent",
			"average_bitrate_kbps",
			"video_playback_failures_percent",
			"ended_plays",
			"connection_induced_rebuffering_ratio_percent",
			"video_restart_time",
		},
		"audience_metriclens": {
			"concurrent_plays",
			"plays",
			"ended_plays",
		},
	}
	if conf.MetricConfigs == nil {
		conf.MetricConfigs = []*MetricConfig{{Metric: "quality_metriclens"}}
	}
	setAccounts(m, conf)
	setFilters(m, conf)
	setDimensions(m, conf)
	setMetriclensFilters(m, conf)
}

func setAccounts(m *Monitor, conf *Config)  {
	jsonResponse, err := get(m, conf, "https://api.conviva.com/insights/2.4/accounts.json")
	if err != nil {
		logger.Errorf("Get accounts request failed %v\n", err)
		return
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

func setFilters(m *Monitor, conf *Config) {
	var jsonResponse   = map[string]map[string]interface{}{}
	var filterIdByName = map[string]map[string]string{}
	if conf.filterNameById == nil {
		conf.filterNameById = make(map[string]map[string]string)
	}
	for _, metricConfig := range conf.MetricConfigs {
		if len(jsonResponse[metricConfig.Account]) == 0 {
			var err error
			url := "https://api.conviva.com/insights/2.4/filters.json?account=" + metricConfig.accountId
			if jsonResponse[metricConfig.Account], err = get(m, conf, url); err != nil {
				logger.Errorf("Failed to get filters for account %s: \n%v\n", metricConfig.Account, err)
				continue
			}
		}
		if len(filterIdByName[metricConfig.Account]) == 0 {
			filterIdByName[metricConfig.Account] = map[string]string{}
		}
		var noFilterNameById bool
		if noFilterNameById = len(conf.filterNameById[metricConfig.Account]) == 0; noFilterNameById {
			conf.filterNameById[metricConfig.Account] = map[string]string{}
		}
		for filterId, filterName := range jsonResponse[metricConfig.Account] {
			if noFilterNameById {
				conf.filterNameById[metricConfig.Account][filterId] = filterName.(string)
				filterIdByName[metricConfig.Account][filterName.(string)] = filterId
			}
		}
		if len(metricConfig.Filters) == 0 {
			metricConfig.Filters   = []string{"All Traffic",}
			metricConfig.filterIds = []string{filterIdByName[metricConfig.Account][metricConfig.Filters[0]],}
		} else if metricConfig.Filters[0] == "*" {
			metricConfig.Filters   = make([]string, 0, len(conf.filterNameById[metricConfig.Account]))
			metricConfig.filterIds = make([]string, 0, len(metricConfig.Filters))
			for filterId, filterName := range conf.filterNameById[metricConfig.Account] {
				metricConfig.Filters   = append(metricConfig.Filters, filterName)
				metricConfig.filterIds = append(metricConfig.filterIds, filterId)
			}
		} else {
			metricConfig.filterIds = make([]string, 0, len(metricConfig.Filters))
			for _, filterName := range metricConfig.Filters {
				metricConfig.filterIds = append(metricConfig.filterIds, filterIdByName[metricConfig.Account][filterName])
			}
		}
	}
}

func setMetriclensFilters(m *Monitor, conf *Config) {
	for _, metricConfig := range conf.MetricConfigs {
		if strings.Contains(metricConfig.Metric, "metriclens") {
			aDimension := metricConfig.Dimensions[0]
			for _, filterId := range metricConfig.filterIds {
				url := "https://api.conviva.com/insights/2.4/metrics.json?metrics=" + metricConfig.Metric +
					"&filter_ids=" + filterId +
					"&metriclens_dimension_id=" + conf.dimensionIdByAccountAndName[metricConfig.Account][aDimension]
				if _, err := get(m, conf, url); err == nil {
					metricConfig.metriclensFilterIds = append(metricConfig.metriclensFilterIds, filterId)
				}
			}
		}
	}
}

func setDimensions(m *Monitor, conf *Config) {
	var jsonResponse = make(map[string]map[string]interface{})
	if conf.dimensionIdByAccountAndName == nil {
		conf.dimensionIdByAccountAndName = make(map[string]map[string]string)
	}
	for _, metricConfig := range conf.MetricConfigs {
		if len(jsonResponse[metricConfig.Account]) == 0 {
			var err error
			jsonResponse[metricConfig.Account], err = get(m, conf, "https://api.conviva.com/insights/2.4/metriclens_dimension_list.json?account=" + metricConfig.accountId)
			if err != nil {
				logger.Errorf("Failed to get metriclens dimensions list for account %s: \n%v\n", metricConfig.Account, err)
				continue
			}
		}
		var noDimensionIdByAccountAndName, noDimensions bool
		if noDimensionIdByAccountAndName = len(conf.dimensionIdByAccountAndName[metricConfig.Account]) == 0; noDimensionIdByAccountAndName {
			conf.dimensionIdByAccountAndName[metricConfig.Account] = make(map[string]string)
		}
		if noDimensions = len(metricConfig.Dimensions) == 0; noDimensions {
			metricConfig.Dimensions = make([]string, 0, len(conf.dimensionIdByAccountAndName[metricConfig.Account]))
		}
		for dimension, dimensionId := range jsonResponse[metricConfig.Account] {
			if noDimensionIdByAccountAndName {
				conf.dimensionIdByAccountAndName[metricConfig.Account][dimension] = strconv.FormatFloat(dimensionId.(float64), 'f', 0, 64)
			}
			if noDimensions {
				metricConfig.Dimensions = append(metricConfig.Dimensions, dimension)
			}
		}
	}
}

