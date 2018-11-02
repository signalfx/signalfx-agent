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

func (m *metricConfig) init(service *accountService) {
	if !m.isInitialized {
		// setting account id and default account if necessary
		if m.Account == "" {
			m.Account = (*service).getDefault().Name
		}
		m.Account = strings.TrimSpace(m.Account)
		m.accountID = (*service).getID(m.Account)
		if m.accountID == "" {
			logger.Errorf("No id for account %s. Wrong account name.", m.Account)
			return
		}
		// setting filter names and filter ids
		if len(m.Filters) == 0 {
			m.Filters   = []string{"All Traffic",}
			m.filterMap = map[string]string{(*service).getFilterID(m.Account, "All Traffic"): "All Traffic",}
		} else if m.Filters[0] == "_ALL_" {
			var allFilters map[string]string
			if strings.Contains(m.Metric, "metriclens") {
				allFilters = (*service).getMetricLensFilters(m.Account)
			} else {
				allFilters = (*service).getFilters(m.Account)
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
				id := (*service).getFilterID(m.Account, name)
				if id == "" {
					logger.Errorf("No id for filter %s. Wrong filter name.", name)
					continue
				}
				m.filterMap[id] = name
			}
		}
		// setting metriclens dimensions
		if strings.Contains(m.Metric, "metriclens") {
			if len(m.MetriclensDimensions) == 0 || m.MetriclensDimensions[0] == "_ALL_" {
				m.MetriclensDimensions   = make([]string, 0, len((*service).getMetricLensDimensionMap(m.Account)))
				m.metriclensDimensionMap = make(map[string]float64, len((*service).getMetricLensDimensionMap(m.Account)))
				for name, id := range (*service).getMetricLensDimensionMap(m.Account) {
					m.MetriclensDimensions = append(m.MetriclensDimensions, name)
					m.metriclensDimensionMap[name] = id
				}
			} else {
				m.metriclensDimensionMap = make(map[string]float64, len(m.MetriclensDimensions))
				for _, name := range m.MetriclensDimensions {
					m.metriclensDimensionMap[name] = (*service).getMetricLensDimensionID(m.Account, name)
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

