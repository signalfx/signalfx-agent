package conviva

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
)

var prefixedMetricLensMetrics = map[string][]string{
	"quality_metriclens": {
		"conviva.quality_metriclens.total_attempts",
		"conviva.quality_metriclens.video_start_failures_percent",
		"conviva.quality_metriclens.exits_before_video_start_percent",
		"conviva.quality_metriclens.plays_percent",
		"conviva.quality_metriclens.video_startup_time_sec",
		"conviva.quality_metriclens.rebuffering_ratio_percent",
		"conviva.quality_metriclens.average_bitrate_kbps",
		"conviva.quality_metriclens.video_playback_failures_percent",
		"conviva.quality_metriclens.ended_plays",
		"conviva.quality_metriclens.connection_induced_rebuffering_ratio_percent",
		"conviva.quality_metriclens.video_restart_time",
	},
	"audience_metriclens": {
		"conviva.audience_metriclens.concurrent_plays",
		"conviva.audience_metriclens.plays",
		"conviva.audience_metriclens.ended_plays",
	},
}

// metricConfig for configuring individual metric
type metricConfig struct {
	// Conviva customer account name. The default account is fetched used if not specified.
	Account string `yaml:"account"`
	Metric  string `yaml:"metric" default:"quality_metriclens"`
	// Filter names. The default is `All Traffic` filter
	Filters []string `yaml:"filters"`
	// MetricLens dimension names. The default is names of all MetricLens dimensions of the account
	MetricLensDimensions []string `yaml:"metricLensDimensions"`
	// MetricLens dimension names to exclude.
	ExcludeMetricLensDimensions []string `yaml:"excludeMetricLensDimensions"`
	accountID                   string
	// id:name map of filters derived from the configured Filters
	filters map[string]string
	// name:id map of MetricLens dimensions derived from configured MetricLensDimensions
	metricLensDimensionMap map[string]float64
	isInitialized          bool
	// id:name map of filters in filters_warmup status on response
	filtersWarmup map[string]string
	// id:name map of filters in filters_not_exist status on response
	filtersNotExist map[string]string
	// id:name map of filters in filters_incomplete_data status on response
	filtersIncompleteData map[string]string
	mutex                 sync.RWMutex
}

func (mc *metricConfig) init(service accountsService) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	if !mc.isInitialized {
		if err := mc.setAccount(service); err != nil {
			logger.Errorf("Metric %s account setting failure. %+v", mc.Metric, err)
			return
		}
		if err := mc.setFilters(service); err != nil {
			logger.Errorf("Metric %s filter(s) setting failure. %+v", mc.Metric, err)
			return
		}
		if err := mc.setMetricLensDimensions(service); err != nil {
			logger.Errorf("Metric %s MetricLens dimension(s) setting failure. %+v", mc.Metric, err)
			return
		}
		if err := mc.excludeMetricLensDimensions(service); err != nil {
			logger.Errorf("Metric %s MetricLens dimension(s) exclusion failure. %+v", mc.Metric, err)
			return
		}
		mc.isInitialized = true
	}
}

// setting account id and default account if necessary
func (mc *metricConfig) setAccount(service accountsService) error {
	mc.Account = strings.TrimSpace(mc.Account)
	if mc.Account == "" {
		if defaultAccount, err := service.getDefault(); err == nil {
			mc.Account = defaultAccount.Name
		} else {
			return err
		}
	}
	var err error
	if mc.accountID, err = service.getID(mc.Account); err != nil {
		return err
	}
	return nil
}

func (mc *metricConfig) setFilters(service accountsService) error {
	if len(mc.Filters) == 0 {
		mc.Filters = []string{"All Traffic"}
		if id, err := service.getFilterID(mc.Account, "All Traffic"); err == nil {
			mc.filters = map[string]string{id: "All Traffic"}
		} else {
			return err
		}
	} else if strings.TrimSpace(mc.Filters[0]) == "_ALL_" {
		var allFilters map[string]string
		var err error
		if strings.Contains(mc.Metric, "metriclens") {
			if allFilters, err = service.getMetricLensFilters(mc.Account); err != nil {
				return err
			}
		} else {
			if allFilters, err = service.getFilters(mc.Account); err != nil {
				return err
			}
		}
		mc.Filters = make([]string, 0, len(allFilters))
		mc.filters = make(map[string]string, len(allFilters))
		for id, name := range allFilters {
			mc.Filters = append(mc.Filters, name)
			mc.filters[id] = name
		}
	} else {
		mc.filters = make(map[string]string, len(mc.Filters))
		for _, name := range mc.Filters {
			name = strings.TrimSpace(name)
			if id, err := service.getFilterID(mc.Account, name); err == nil {
				mc.filters[id] = name
			} else {
				return err
			}
		}
	}
	return nil
}

func (mc *metricConfig) setMetricLensDimensions(service accountsService) error {
	if strings.Contains(mc.Metric, "metriclens") {
		if len(mc.MetricLensDimensions) == 0 || strings.TrimSpace(mc.MetricLensDimensions[0]) == "_ALL_" {
			if metricLensDimensionMap, err := service.getMetricLensDimensionMap(mc.Account); err == nil {
				mc.MetricLensDimensions = make([]string, 0, len(metricLensDimensionMap))
				mc.metricLensDimensionMap = make(map[string]float64, len(metricLensDimensionMap))
				for name, id := range metricLensDimensionMap {
					mc.MetricLensDimensions = append(mc.MetricLensDimensions, name)
					mc.metricLensDimensionMap[name] = id
				}
			} else {
				return err
			}

		} else {
			mc.metricLensDimensionMap = make(map[string]float64, len(mc.MetricLensDimensions))
			for i, name := range mc.MetricLensDimensions {
				name := strings.TrimSpace(name)
				if id, err := service.getMetricLensDimensionID(mc.Account, name); err == nil {
					mc.MetricLensDimensions[i] = name
					mc.metricLensDimensionMap[name] = id
				} else {
					return err
				}
			}
		}
	}
	return nil
}

func (mc *metricConfig) excludeMetricLensDimensions(service accountsService) error {
	for _, excludeName := range mc.ExcludeMetricLensDimensions {
		excludeName := strings.TrimSpace(excludeName)
		if _, err := service.getMetricLensDimensionID(mc.Account, excludeName); err == nil {
			delete(mc.metricLensDimensionMap, excludeName)
		} else {
			return err
		}
	}
	if len(mc.metricLensDimensionMap) < len(mc.MetricLensDimensions) {
		mc.MetricLensDimensions = make([]string, 0, len(mc.metricLensDimensionMap))
		for name := range mc.metricLensDimensionMap {
			mc.MetricLensDimensions = append(mc.MetricLensDimensions, name)
		}
	}
	return nil
}

func (mc *metricConfig) filterIDs() []string {
	ids := make([]string, 0, len(mc.filters))
	for id := range mc.filters {
		ids = append(ids, id)
	}
	return ids
}

// logs filter status only when the filter status changes
func (mc *metricConfig) logFilterStatuses(filtersWarmupIds []float64, filtersNotExistIds []float64, filtersIncompleteDataIds []float64) {
	mc.filtersWarmup = logFilterStatusesHelper(mc.Metric, mc.filters, mc.filtersWarmup, filtersWarmupIds, "filters_warmup")
	mc.filtersNotExist = logFilterStatusesHelper(mc.Metric, mc.filters, mc.filtersNotExist, filtersNotExistIds, "filters_not_exist")
	mc.filtersIncompleteData = logFilterStatusesHelper(mc.Metric, mc.filters, mc.filtersIncompleteData, filtersIncompleteDataIds, "filters_incomplete_data")
}

func logFilterStatusesHelper(metric string, filters map[string]string, filterStatusesCurrent map[string]string, filterStatusesIDsNew []float64, status string) map[string]string {
	filterStatusesNew := map[string]string{}
	filterStatusesToLog := map[string]string{}
	if filterStatusesCurrent == nil {
		filterStatusesCurrent = map[string]string{}
	}
	for _, id := range filterStatusesIDsNew {
		id := strconv.FormatFloat(id, 'f', 0, 64)
		if filterStatusesCurrent[id] == "" {
			filterStatusesToLog[id] = filters[id]
		} else {
			delete(filterStatusesCurrent, id)
		}
		filterStatusesNew[id] = filters[id]
	}
	if len(filterStatusesToLog) != 0 {
		if m, err := json.Marshal(filterStatusesToLog); err == nil {
			logger.Warnf("GET metric %s has filters in the unresponsive %s status. Set log level to debug to see status change to responsive in future requests. Filters in %s status: %s", metric, status, status, m)
		} else {
			logger.Errorf("Failed marshalling id:name map of filters in %s status. %+v", status, err)
		}
	}
	if len(filterStatusesCurrent) != 0 {
		if m, err := json.Marshal(filterStatusesCurrent); err == nil {
			logger.Debugf("GET metric %s has filters whose status changed from %s to responsive. Filters whose status changed: %s", metric, status, m)
		} else {
			logger.Errorf("Failed marshalling id:name map of filters out of %s status. %+v", status, err)
		}
	}
	return filterStatusesNew
}
