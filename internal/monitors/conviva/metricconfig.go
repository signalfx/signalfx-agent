package conviva

import (
	"strings"
)

var metriclensMetrics = map[string][]string {
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

// MetricConfig for configuring individual metric
type MetricConfig struct {
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

func (m *MetricConfig) init(accountService *AccountService) {
	if !m.isInitialized {
		// setting account id and default account if necessary
		if m.Account == "" {
			m.Account = (*accountService).GetDefault().Name
		}
		m.Account = strings.TrimSpace(m.Account)
		m.accountID = (*accountService).GetID(m.Account)
		if m.accountID == "" {
			logger.Errorf("No id for account %s. Wrong account name.", m.Account)
			return
		}
		// setting filter names and filter ids
		if len(m.Filters) == 0 {
			m.Filters   = []string{"All Traffic",}
			m.filterMap = map[string]string{(*accountService).GetFilterID(m.Account, "All Traffic"): "All Traffic",}
		} else if m.Filters[0] == "_ALL_" {
			var allFilters map[string]string
			if strings.Contains(m.Metric, "metriclens") {
				allFilters = (*accountService).GetMetricLensFilters(m.Account)
			} else {
				allFilters = (*accountService).GetFilters(m.Account)
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
				id := (*accountService).GetFilterID(m.Account, name)
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
				m.MetriclensDimensions   = make([]string, 0, len((*accountService).GetMetricLensDimensionMap(m.Account)))
				m.metriclensDimensionMap = make(map[string]float64, len((*accountService).GetMetricLensDimensionMap(m.Account)))
				for name, id := range (*accountService).GetMetricLensDimensionMap(m.Account) {
					m.MetriclensDimensions = append(m.MetriclensDimensions, name)
					m.metriclensDimensionMap[name] = id
				}
			} else {
				m.metriclensDimensionMap = make(map[string]float64, len(m.MetriclensDimensions))
				for _, name := range m.MetriclensDimensions {
					m.metriclensDimensionMap[name] = (*accountService).GetMetricLensDimensionID(m.Account, name)
				}
			}
		}
		m.isInitialized = true
	}
}

func (m *MetricConfig) filterIDs() []string {
	ids := make([]string, 0, len(m.filterMap))
	for id := range m.filterMap {
		ids = append(ids, id)
	}
	return ids
}

