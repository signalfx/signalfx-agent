package conviva

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
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

func initConfig(m *Monitor, conf *Config)  {
	if conf.MetricConfigs == nil {
		conf.MetricConfigs = []*MetricConfig{{Metric: "quality_metriclens"}}
	}
	initAccounts(m, conf)
	initFilters(m, conf)
	initMetricLensDimensions(m, conf)
	initMetricLensFilters(m, conf)
	initMetricConfigs(m, conf)
}

func initAccounts(m *Monitor, conf *Config) error {
	ctx, cancel := context.WithTimeout(m.ctx, time.Duration(conf.TimeoutSeconds) * time.Second)
	defer cancel()
	if conf.defaultAccount == "" {
		var res map[string]interface{}; var err error
		if res, err = get(ctx, m, conf, "https://api.conviva.com/insights/2.4/accounts.json"); err != nil {
			logger.Errorf("GET account(s) failed. %v", err)
			return err
		}
		if res["accounts"] == nil {
			return fmt.Errorf("response for get accounts returned no account")
		}
		if res["default"] != nil {
			conf.defaultAccount = res["default"].(string)
		}
		conf.accounts = make(map[string]string, len(res["accounts"].(map[string]interface{})))
		for name, id := range res["accounts"].(map[string]interface{}) {
			if id != nil {
				conf.accounts[name] = id.(string)
			}
		}
	}
	return nil
}

func initFilters(m *Monitor, conf *Config) {
	if len(conf.filterByAccountAndID) == 0 || len(conf.filterIDByAccountAndName) == 0 {
		conf.filterByAccountAndID = map[string]map[string]string{}
		conf.filterIDByAccountAndName = map[string]map[string]string{}
	}
	var (
		waitGroup sync.WaitGroup
		mutex  sync.RWMutex
	)
	for account, accountID := range conf.accounts {
		if len(conf.filterByAccountAndID[account]) == 0 || len(conf.filterIDByAccountAndName[account]) == 0 {
			waitGroup.Add(1)
			go func(account string, accountID string) {
				ctx, cancel := context.WithTimeout(m.ctx, time.Duration(conf.TimeoutSeconds)*time.Second)
				defer waitGroup.Done()
				defer cancel()
				if res, err := get(ctx, m, conf, "https://api.conviva.com/insights/2.4/filters.json?account="+accountID); err != nil {
					logger.Errorf("GET filters for account %s failed. %v", account, err)
				} else {
					mutex.Lock()
					conf.filterByAccountAndID[account]     = make(map[string]string, len(res))
					conf.filterIDByAccountAndName[account] = make(map[string]string, len(res))
					mutex.Unlock()
					for id, name := range res {
						if name != nil {
							mutex.Lock()
							conf.filterByAccountAndID[account][id]                = name.(string)
							conf.filterIDByAccountAndName[account][name.(string)] = id
							mutex.Unlock()
						}
					}
				}
			}(account, accountID)
		}
	}
	waitGroup.Wait()
}

func initMetricLensDimensions(m *Monitor, conf *Config)  {
	if len(conf.metriclensDimensionIDByAccountAndName) == 0 {
		conf.metriclensDimensionIDByAccountAndName = map[string]map[string]string{}
	}
	var (
		waitGroup sync.WaitGroup
		mutex  sync.RWMutex
	)
	for account, accountID := range conf.accounts {
		if len(conf.metriclensDimensionIDByAccountAndName[account]) == 0 {
			waitGroup.Add(1)
			go func(account string, accountID string) {
				ctx, cancel := context.WithTimeout(m.ctx, time.Duration(conf.TimeoutSeconds)*time.Second)
				defer waitGroup.Done()
				defer cancel()
				if res, err := get(ctx, m, conf, "https://api.conviva.com/insights/2.4/metriclens_dimension_list.json?account="+accountID); err != nil {
					logger.Errorf("GET metriclens dimensions list for account %s failed. %v", account, err)
				} else {
					mutex.Lock()
					conf.metriclensDimensionIDByAccountAndName[account] = make(map[string]string, len(res))
					mutex.Unlock()
					for name, id := range res {
						if id != nil {
							id := strconv.FormatFloat(id.(float64), 'f', 0, 64)
							mutex.Lock()
							conf.metriclensDimensionIDByAccountAndName[account][name] = id
							mutex.Unlock()
						}
					}
				}
			}(account, accountID)
		}
	}
	waitGroup.Wait()
}

func initMetricLensFilters(m *Monitor, conf *Config) {
	if len(conf.metricLensFilterIDByAccountAndName) == 0 {
		conf.metricLensFilterIDByAccountAndName = map[string]map[string]string{}
	}
	var (
		waitGroup sync.WaitGroup
		mutex  sync.RWMutex
	)
	for account, accountID := range conf.accounts {
		if len(conf.metricLensFilterIDByAccountAndName[account]) == 0 {
			conf.metricLensFilterIDByAccountAndName[account] = make(map[string]string)
		}
		for filterID, filter := range conf.filterByAccountAndID[account] {
			waitGroup.Add(1)
			go func(accountID string, account string, filterID string, filter string) {
				ctx, cancel := context.WithTimeout(m.ctx, time.Duration(conf.TimeoutSeconds)*time.Second)
				defer waitGroup.Done()
				defer cancel()
				for _, aDimensionID := range conf.metriclensDimensionIDByAccountAndName[account] {
					if _, err := get(ctx, m, conf, "https://api.conviva.com/insights/2.4/metrics.json?metrics=audience_metriclens&account="+accountID+"&filter_ids="+filterID+"&metriclens_dimension_id="+aDimensionID); err == nil {
						mutex.Lock()
						conf.metricLensFilterIDByAccountAndName[account][filter] = filterID
						mutex.Unlock()
					}
					break
				}
			}(accountID, account, filterID, filter)
		}
	}
	waitGroup.Wait()
}

func initMetricConfigs(m *Monitor, conf *Config) {
	for _, metricConfig := range conf.MetricConfigs {
		if metricConfig.Account == "" {
			metricConfig.Account = conf.defaultAccount
		}
		metricConfig.Account = strings.TrimSpace(metricConfig.Account)
		metricConfig.accountID = conf.accounts[metricConfig.Account]
		if metricConfig.accountID == "" {
			logger.Errorf("No id for account %s. Wrong account name.", metricConfig.Account)
			continue
		}
		filterIDByAccountAndName := conf.filterIDByAccountAndName
		if strings.Contains(metricConfig.Metric, "metriclens") {
			filterIDByAccountAndName = conf.metricLensFilterIDByAccountAndName
			if len(metricConfig.MetriclensDimensions) == 0 || metricConfig.MetriclensDimensions[0] == "_ALL_" {
				metricConfig.MetriclensDimensions = make([]string, 0, len(conf.metriclensDimensionIDByAccountAndName[metricConfig.Account]))
				for dimension := range conf.metriclensDimensionIDByAccountAndName[metricConfig.Account] {
					metricConfig.MetriclensDimensions = append(metricConfig.MetriclensDimensions, dimension)
				}
			}
		}
		if len(metricConfig.Filters) == 0 {
			metricConfig.Filters   = []string{"All Traffic",}
			metricConfig.filterIDs = []string{filterIDByAccountAndName[metricConfig.Account][metricConfig.Filters[0]],}
		} else if metricConfig.Filters[0] == "_ALL_" {
			metricConfig.filterIDs = make([]string, 0, len(filterIDByAccountAndName[metricConfig.Account]))
			metricConfig.Filters = make([]string, 0, len(metricConfig.filterIDs))
			for name, id := range filterIDByAccountAndName[metricConfig.Account] {
				metricConfig.Filters = append(metricConfig.Filters, name)
				metricConfig.filterIDs = append(metricConfig.filterIDs, id)
			}
		} else {
			metricConfig.filterIDs = make([]string, 0, len(metricConfig.Filters))
			for _, name := range metricConfig.Filters {
				name = strings.TrimSpace(name)
				if filterIDByAccountAndName[metricConfig.Account][name] == "" {
					logger.Errorf("No id for filter %s. Wrong filter name.", name)
					continue
				}
				metricConfig.filterIDs = append(metricConfig.filterIDs, filterIDByAccountAndName[metricConfig.Account][name])
			}
		}
	}
}