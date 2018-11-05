package conviva

import (
	"strings"
)

var prefixedMetriclensMetrics = map[string][]string {
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
	// Conviva customer account name. The default account is used if not specified.
	Account                string   `yaml:"account"`
	Metric                 string   `yaml:"metric" default:"quality_metriclens"`
	// Filter names. The default is `All Traffic` filter
	Filters                []string `yaml:"filters"`
	// Metriclens dimension names.
	MetriclensDimensions   []string `yaml:"metriclensDimensions"`
	accountID              string
	filterMap              map[string]string
	metriclensDimensionMap map[string]float64
	isInitialized          bool
}

func (m *metricConfig) init(service accountsService) {
	if !m.isInitialized {
		// setting account id and default account if necessary
		if m.Account == ""  {
			if defaultAccount, err := service.getDefault(); err != nil {
				logger.Error(err)
				return
			} else {
				m.Account = defaultAccount.Name
			}
		}
		m.Account = strings.TrimSpace(m.Account)
		var err error
		if m.accountID, err = service.getID(m.Account); err != nil {
			logger.Error(err)
			return
		}
		if len(m.Filters) == 0 {
			m.Filters   = []string{"All Traffic",}
			if id, err := service.getFilterID(m.Account, "All Traffic"); err == nil {
				m.filterMap = map[string]string{id: "All Traffic",}
			} else {
				logger.Error(err)
				return
			}
		} else if m.Filters[0] == "_ALL_" {
			var allFilters map[string]string
			var err error
			if strings.Contains(m.Metric, "metriclens") {
				if allFilters, err = service.getMetricLensFilters(m.Account); err != nil {
					logger.Error(err)
					return
				}
			} else {
				if allFilters, err = service.getFilters(m.Account); err != nil {
					logger.Error(err)
					return
				}
			}
			m.Filters   = make([]string, 0, len(allFilters))
			m.filterMap = make(map[string]string, len(allFilters))
			for id, name := range allFilters {
				m.Filters = append(m.Filters, name)
				m.filterMap[id] = name
			}
		} else {
			m.filterMap = make(map[string]string, len(m.Filters))
			for _, name := range m.Filters {
				name = strings.TrimSpace(name)
				if id, err := service.getFilterID(m.Account, name); err == nil {
					m.filterMap[id] = name
				} else {
					logger.Error(err)
					return
				}
			}
		}
		// setting metriclens dimensions
		if strings.Contains(m.Metric, "metriclens") {
			if len(m.MetriclensDimensions) == 0 || m.MetriclensDimensions[0] == "_ALL_" {
				if metricLensDimensionMap, err := service.getMetricLensDimensionMap(m.Account); err == nil {
					m.MetriclensDimensions = make([]string, 0, len(metricLensDimensionMap))
					m.metriclensDimensionMap = make(map[string]float64, len(metricLensDimensionMap))
					for name, id := range metricLensDimensionMap {
						m.MetriclensDimensions = append(m.MetriclensDimensions, name)
						m.metriclensDimensionMap[name] = id
					}
				} else {
					logger.Error(err)
					return
				}

			} else {
				m.metriclensDimensionMap = make(map[string]float64, len(m.MetriclensDimensions))
				for _, name := range m.MetriclensDimensions {
					if id, err := service.getMetricLensDimensionID(m.Account, name); err == nil {
						m.metriclensDimensionMap[name] = id
					} else {
						logger.Error(err)
						return
					}
				}
			}
		}
		m.isInitialized = true
	}
}

func (m *metricConfig) filterIDs() []string {
	ids := make([]string, 0, len(m.filterMap))
	for id := range m.filterMap {
		ids = append(ids, id)
	}
	return ids
}

