package conviva

import (
	"strconv"
	"strings"
)

func setConfigFields(m *Monitor, conf *Config)  {
	conf.metriclensMetrics = map[string][]string{
		"quality_metriclens": {
			"Total Attempts",
			"Video Start Failures (%)",
			"Exits Before Video Start (%)",
			"Plays (%)",
			"Video Startup Time (sec)",
			"Rebuffering Ratio (%)",
			"Average Bitrate (kbps)",
			"Video Playback Failures (%)",
			"Ended Plays",
			"Connection Induced Rebuffering Ratio (%)",
			"Video Restart Time",
		},
		"audience_metriclens": {
			"Concurrent Plays",
			"Plays",
			"Ended Plays",
		},
	}
	if conf.MetricConfigs == nil {
		conf.MetricConfigs = []*MetricConfig{{Metric: "quality_metriclens"}}
	}
	setAccounts(m, conf)
	setFilters(m, conf)
	setMetriclensDimensions(m, conf)
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
					metricConfig.accountID = id.(string)
				}
			}
		}
	}
}

func setFilters(m *Monitor, conf *Config) {
	var jsonResponse   = map[string]map[string]interface{}{}
	var filterIDByName = map[string]map[string]string{}
	if conf.filterNameByID == nil {
		conf.filterNameByID = make(map[string]map[string]string)
	}
	for _, metricConfig := range conf.MetricConfigs {
		if len(jsonResponse[metricConfig.Account]) == 0 {
			var err error
			url := "https://api.conviva.com/insights/2.4/filters.json?account=" + metricConfig.accountID
			if jsonResponse[metricConfig.Account], err = get(m, conf, url); err != nil {
				logger.Errorf("Failed to get filters for account %s: \n%v\n", metricConfig.Account, err)
				continue
			}
		}
		if len(filterIDByName[metricConfig.Account]) == 0 {
			filterIDByName[metricConfig.Account] = map[string]string{}
		}
		var noFilterNameByID bool
		if noFilterNameByID = len(conf.filterNameByID[metricConfig.Account]) == 0; noFilterNameByID {
			conf.filterNameByID[metricConfig.Account] = map[string]string{}
		}
		for filterID, filterName := range jsonResponse[metricConfig.Account] {
			if noFilterNameByID {
				conf.filterNameByID[metricConfig.Account][filterID] = filterName.(string)
				filterIDByName[metricConfig.Account][filterName.(string)] = filterID
			}
		}
		if len(metricConfig.Filters) == 0 {
			metricConfig.Filters   = []string{"All Traffic",}
			metricConfig.filterIDs = []string{filterIDByName[metricConfig.Account][metricConfig.Filters[0]],}
		} else if metricConfig.Filters[0] == "_ALL_" {
			metricConfig.Filters   = make([]string, 0, len(conf.filterNameByID[metricConfig.Account]))
			metricConfig.filterIDs = make([]string, 0, len(metricConfig.Filters))
			for filterID, filterName := range conf.filterNameByID[metricConfig.Account] {
				metricConfig.Filters   = append(metricConfig.Filters, filterName)
				metricConfig.filterIDs = append(metricConfig.filterIDs, filterID)
			}
		} else {
			metricConfig.filterIDs = make([]string, 0, len(metricConfig.Filters))
			for _, filterName := range metricConfig.Filters {
				metricConfig.filterIDs = append(metricConfig.filterIDs, filterIDByName[metricConfig.Account][filterName])
			}
		}
	}
}

func setMetriclensFilters(m *Monitor, conf *Config) {
	for _, metricConfig := range conf.MetricConfigs {
		if strings.Contains(metricConfig.Metric, "metriclens") {
			aDimension := metricConfig.MetriclensDimensions[0]
			for _, filterID := range metricConfig.filterIDs {
				url := "https://api.conviva.com/insights/2.4/metrics.json?metrics=" + metricConfig.Metric +
					"&filter_ids=" + filterID +
					"&metriclens_dimension_id=" + conf.metriclensDimensionIDByAccountAndName[metricConfig.Account][aDimension]
				if _, err := get(m, conf, url); err == nil {
					metricConfig.metriclensFilterIDs = append(metricConfig.metriclensFilterIDs, filterID)
				}
			}
		}
	}
}

func setMetriclensDimensions(m *Monitor, conf *Config) {
	var metriclensDimensionsResponse = make(map[string]map[string]interface{})
	if conf.metriclensDimensionIDByAccountAndName == nil {
		conf.metriclensDimensionIDByAccountAndName = make(map[string]map[string]string)
	}
	for _, metricConfig := range conf.MetricConfigs {
		if len(metriclensDimensionsResponse[metricConfig.Account]) == 0 {
			var err error
			metriclensDimensionsResponse[metricConfig.Account], err = get(m, conf, "https://api.conviva.com/insights/2.4/metriclens_dimension_list.json?account=" + metricConfig.accountID)
			if err != nil {
				logger.Errorf("Failed to get metriclens dimensions list for account %s: \n%v\n", metricConfig.Account, err)
				continue
			}
		}
		var noMetriclensDimensionIDByAccountAndName, noMetriclensDimensions bool
		if noMetriclensDimensionIDByAccountAndName = len(conf.metriclensDimensionIDByAccountAndName[metricConfig.Account]) == 0; noMetriclensDimensionIDByAccountAndName {
			conf.metriclensDimensionIDByAccountAndName[metricConfig.Account] = make(map[string]string)
		}
		if noMetriclensDimensions = len(metricConfig.MetriclensDimensions) == 0 || metricConfig.MetriclensDimensions[0] == "_ALL_"; noMetriclensDimensions {
			metricConfig.MetriclensDimensions = make([]string, 0, len(metriclensDimensionsResponse[metricConfig.Account]))
		}
		for metriclensDimension, metriclensDimensionID := range metriclensDimensionsResponse[metricConfig.Account] {
			if noMetriclensDimensionIDByAccountAndName {
				conf.metriclensDimensionIDByAccountAndName[metricConfig.Account][metriclensDimension] = strconv.FormatFloat(metriclensDimensionID.(float64), 'f', 0, 64)
			}
			if noMetriclensDimensions {
				metricConfig.MetriclensDimensions = append(metricConfig.MetriclensDimensions, metriclensDimension)
			}
		}
	}
}

