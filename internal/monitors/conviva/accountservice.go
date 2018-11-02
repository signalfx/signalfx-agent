package conviva

import (
	"context"
	"fmt"
	"sync"
	"time"
)

//const (
//	metricLensURLFormat = "https://api.conviva.com/insights/2.4/metrics.json?metrics=%s&account=%s&filter_ids=%s&metriclens_dimension_id=%d"
//)

var anyErrors bool
var isInitialized bool

// Account for Conviva account data
type account struct {
	id                   string
	Name                 string
	filters              map[string]string
	metricLensFilters    map[string]string
	metricLensDimensions map[string]float64
}

// AccountService interface for Account related methods
type accountService interface {
	getDefault()                                                             *account
	getMetricLensDimensionMap(accountName string)                            map[string]float64
	getID(accountName string)                                                string
	getFilters(accountName string)                                           map[string]string
	getMetricLensFilters(accountName string)                                 map[string]string
	getFilterID(accountName string, filterName string)                       string
	getMetricLensDimensionID(accountName string, metricLensDimension string) float64
}

type accountServiceImpl struct {
	defaultAccount *account
	accounts       []*account
	ctx            context.Context
	timeout        *time.Duration
	client         *httpClient
	mutex          *sync.RWMutex
}

// NewAccountService factory function creating AccountService
func newAccountService(ctx context.Context, timeout *time.Duration, client *httpClient) accountService {
	service := accountServiceImpl{ctx: ctx, timeout: timeout, client: client, mutex: &sync.RWMutex{},}
	return &service
}

func (s *accountServiceImpl) getDefault() *account {
	s.init()
	return s.defaultAccount
}

func (s *accountServiceImpl) getMetricLensDimensionMap(accountName string) map[string]float64 {
	s.init()
	for _, a := range s.accounts {
		if a.Name == accountName {
			return a.metricLensDimensions
		}
	}
	return nil
}

func (s *accountServiceImpl) getID(accountName string) string {
	if len(s.accounts) == 0 {
		s.init()
	}
	for _, a := range s.accounts {
		if a.Name == accountName {
			return a.id
		}
	}
	return ""
}

func (s *accountServiceImpl) getFilters(accountName string) map[string]string {
	s.init()
	for _, a := range s.accounts {
		if a.Name == accountName {
			return a.filters
		}
	}
	return nil
}

func (s *accountServiceImpl) getMetricLensFilters(accountName string) map[string]string {
	s.init()
	for _, a := range s.accounts {
		if a.Name == accountName {
			return a.metricLensFilters
		}
	}
	return nil
}

func (s *accountServiceImpl) getFilterID(accountName string, filterName string) string {
	s.init()
	for _, a := range s.accounts {
		if a.Name == accountName {
			for id, name := range a.filters {
				if name == filterName {
					return id
				}
			}
		}
	}
	return ""
}

func (s *accountServiceImpl) getMetricLensDimensionID(accountName string, metricLensDimension string) float64 {
	s.init()
	for _, a := range s.accounts {
		if a.Name == accountName {
			for name, id := range a.metricLensDimensions {
				if name == metricLensDimension {
					return id
				}
			}
		}
	}
	return 0
}

func (s *accountServiceImpl) init() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !isInitialized {
		defer func() {anyErrors = false}()
		ctx, cancel := context.WithTimeout(s.ctx, *s.timeout)
		defer cancel()
		res := struct {
			Default  string            `json:"default"`
			Count    float64           `json:"count"`
			Accounts map[string]string `json:"accounts"`
		}{}
		if err := (*s.client).Get(ctx, &res, "https://api.conviva.com/insights/2.4/accounts.json"); err != nil {
			logger.Error(err)
			return
		}
		s.accounts = make([]*account, 0, len(res.Accounts))
		for name, id := range res.Accounts {
			a := account{Name: name, id: id,}
			if a.Name == res.Default {
				s.defaultAccount = &a
			}
			s.accounts = append(s.accounts, &a)
		}
		s.initFilters()
		s.initMetriclensDimensions()
		s.initMetricLensFilters()
		if !anyErrors {
			isInitialized = true
		}
	}
}

func (s *accountServiceImpl) initFilters() {
	var (
		waitGroup sync.WaitGroup
		mutex  sync.RWMutex
	)
	for _, a := range s.accounts {
		a.filters = map[string]string{}
		waitGroup.Add(1)
		go func(a *account) {
			ctx, cancel := context.WithTimeout(s.ctx, *s.timeout)
			defer waitGroup.Done()
			defer cancel()
			if err := (*s.client).Get(ctx, &a.filters, "https://api.conviva.com/insights/2.4/filters.json?account="+a.id); err != nil {
				logger.Error(err)
				mutex.Lock(); anyErrors = true; mutex.Unlock()
			}
		}(a)
	}
	waitGroup.Wait()
}

func (s *accountServiceImpl) initMetriclensDimensions() {
	var (
		waitGroup sync.WaitGroup
		mutex  sync.RWMutex
	)
	for _, a := range s.accounts {
		a.metricLensDimensions = map[string]float64{}
		waitGroup.Add(1)
		go func(a *account) {
			ctx, cancel := context.WithTimeout(s.ctx, *s.timeout)
			defer waitGroup.Done()
			defer cancel()
			if err := (*s.client).Get(ctx, &a.metricLensDimensions, "https://api.conviva.com/insights/2.4/metriclens_dimension_list.json?account="+a.id); err != nil {
				logger.Error(err)
				mutex.Lock(); anyErrors = true; mutex.Unlock()
			}
		}(a)
	}
	waitGroup.Wait()
}

func (s *accountServiceImpl) initMetricLensFilters() {
	var (
		waitGroup sync.WaitGroup
		mutex  sync.RWMutex
	)
	for _, a := range s.accounts {
		a.metricLensFilters = map[string]string{}
		for filterID, filter := range a.filters {
			waitGroup.Add(1)
			go func(a *account, filterID string, filter string) {
				ctx, cancel := context.WithTimeout(s.ctx, *s.timeout)
				defer waitGroup.Done()
				defer cancel()
				for _, dimID := range a.metricLensDimensions   {
					if err := (*s.client).Get(ctx, &map[string]interface{}{}, fmt.Sprintf(metricLensURLFormat, "quality_metriclens", a.id, filterID, int(dimID))); err == nil {
						mutex.Lock(); a.metricLensFilters[filterID] = filter; mutex.Unlock()
					}
					break
				}
			}(a, filterID, filter)
		}
	}
	waitGroup.Wait()
}
