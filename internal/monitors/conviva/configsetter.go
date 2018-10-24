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
		} else if metricConfig.Filters[0] == "*" {
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
			aDimension := metricConfig.Dimensions[0]
			for _, filterID := range metricConfig.filterIDs {
				url := "https://api.conviva.com/insights/2.4/metrics.json?metrics=" + metricConfig.Metric +
					"&filter_ids=" + filterID +
					"&metriclens_dimension_id=" + conf.dimensionIDByAccountAndName[metricConfig.Account][aDimension]
				if _, err := get(m, conf, url); err == nil {
					metricConfig.metriclensFilterIDs = append(metricConfig.metriclensFilterIDs, filterID)
				}
			}
		}
	}
}

func setDimensions(m *Monitor, conf *Config) {
	var jsonResponse = make(map[string]map[string]interface{})
	if conf.dimensionIDByAccountAndName == nil {
		conf.dimensionIDByAccountAndName = make(map[string]map[string]string)
	}
	for _, metricConfig := range conf.MetricConfigs {
		if len(jsonResponse[metricConfig.Account]) == 0 {
			var err error
			jsonResponse[metricConfig.Account], err = get(m, conf, "https://api.conviva.com/insights/2.4/metriclens_dimension_list.json?account=" + metricConfig.accountID)
			if err != nil {
				logger.Errorf("Failed to get metriclens dimensions list for account %s: \n%v\n", metricConfig.Account, err)
				continue
			}
		}
		var noDimensionIDByAccountAndName, noDimensions bool
		if noDimensionIDByAccountAndName = len(conf.dimensionIDByAccountAndName[metricConfig.Account]) == 0; noDimensionIDByAccountAndName {
			conf.dimensionIDByAccountAndName[metricConfig.Account] = make(map[string]string)
		}
		if noDimensions = len(metricConfig.Dimensions) == 0; noDimensions {
			metricConfig.Dimensions = make([]string, 0, len(conf.dimensionIDByAccountAndName[metricConfig.Account]))
		}
		for dimension, dimensionID := range jsonResponse[metricConfig.Account] {
			if noDimensionIDByAccountAndName {
				conf.dimensionIDByAccountAndName[metricConfig.Account][dimension] = strconv.FormatFloat(dimensionID.(float64), 'f', 0, 64)
			}
			if noDimensions {
				metricConfig.Dimensions = append(metricConfig.Dimensions, dimension)
			}
		}
	}
}

