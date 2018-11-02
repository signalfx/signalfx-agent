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
type Account struct {
	id                   string
	Name                 string
	filters              map[string]string
	metricLensFilters    map[string]string
	metricLensDimensions map[string]float64
}

// AccountService interface for Account related methods
type AccountService interface {
	GetDefault()                                  *Account
	GetMetricLensDimensionMap(accountName string) map[string]float64
	GetID(accountName string)                     string
	GetFilters(accountName string)                map[string]string
	GetMetricLensFilters(accountName string)      map[string]string
	GetFilterID(accountName string, filterName string) string
	GetMetricLensDimensionID(accountName string, metricLensDimension string) float64
}

type accountServiceImpl struct {
	defaultAccount *Account
	accounts       []*Account
	ctx            context.Context
	timeout        *time.Duration
	httpClient     *HTTPClient
}

// NewAccountService factory function creating AccountService
func NewAccountService(ctx context.Context, timeout *time.Duration, httpClient *HTTPClient) AccountService {
	service := accountServiceImpl{ctx: ctx, timeout: timeout, httpClient: httpClient,}
	service.init()
	return &service
}

func (s *accountServiceImpl) GetDefault() *Account {
	s.init()
	return s.defaultAccount
}

func (s *accountServiceImpl) GetMetricLensDimensionMap(accountName string) map[string]float64 {
	s.init()
	for _, account := range s.accounts {
		if account.Name == accountName {
			return account.metricLensDimensions
		}
	}
	return nil
}

func (s *accountServiceImpl) GetID(accountName string) string {
	if len(s.accounts) == 0 {
		s.init()
	}
	for _, account := range s.accounts {
		if account.Name == accountName {
			return account.id
		}
	}
	return ""
}

func (s *accountServiceImpl) GetFilters(accountName string) map[string]string {
	s.init()
	for _, account := range s.accounts {
		if account.Name == accountName {
			return account.filters
		}
	}
	return nil
}

func (s *accountServiceImpl) GetMetricLensFilters(accountName string) map[string]string {
	s.init()
	for _, account := range s.accounts {
		if account.Name == accountName {
			return account.metricLensFilters
		}
	}
	return nil
}

func (s *accountServiceImpl) GetFilterID(accountName string, filterName string) string {
	s.init()
	for _, account := range s.accounts {
		if account.Name == accountName {
			for id, name := range account.filters {
				if name == filterName {
					return id
				}
			}
		}
	}
	return ""
}

func (s *accountServiceImpl) GetMetricLensDimensionID(accountName string, metricLensDimension string) float64 {
	s.init()
	for _, account := range s.accounts {
		if account.Name == accountName {
			for name, id := range account.metricLensDimensions {
				if name == metricLensDimension {
					return id
				}
			}
		}
	}
	return 0
}

func (s *accountServiceImpl) init() {
	defer func() {anyErrors = false}()
	ctx, cancel := context.WithTimeout(s.ctx, *s.timeout)
	defer cancel()
	if !isInitialized {
		res := struct {
			Default  string            `json:"default"`
			Count    float64           `json:"count"`
			Accounts map[string]string `json:"accounts"`
		}{}
		if err := (*s.httpClient).Get(ctx, &res, "https://api.conviva.com/insights/2.4/accounts.json"); err != nil {
			logger.Error(err)
			return
		}
		s.accounts = make([]*Account, 0, len(res.Accounts))
		for name, id := range res.Accounts {
			account := Account{Name: name, id: id,}
			if account.Name == res.Default {
				s.defaultAccount = &account
			}
			s.accounts = append(s.accounts, &account)
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
	for _, account := range s.accounts {
		account.filters = map[string]string{}
		waitGroup.Add(1)
		go func(account *Account) {
			ctx, cancel := context.WithTimeout(s.ctx, *s.timeout)
			defer waitGroup.Done()
			defer cancel()
			if err := (*s.httpClient).Get(ctx, &account.filters, "https://api.conviva.com/insights/2.4/filters.json?account="+account.id); err != nil {
				logger.Error(err)
				mutex.Lock(); anyErrors = true; mutex.Unlock()
			}
		}(account)
	}
	waitGroup.Wait()
}

func (s *accountServiceImpl) initMetriclensDimensions() {
	var (
		waitGroup sync.WaitGroup
		mutex  sync.RWMutex
	)
	for _, account := range s.accounts {
		account.metricLensDimensions = map[string]float64{}
		waitGroup.Add(1)
		go func(account *Account) {
			ctx, cancel := context.WithTimeout(s.ctx, *s.timeout)
			defer waitGroup.Done()
			defer cancel()
			if err := (*s.httpClient).Get(ctx, &account.metricLensDimensions, "https://api.conviva.com/insights/2.4/metriclens_dimension_list.json?account="+account.id); err != nil {
				logger.Error(err)
				mutex.Lock(); anyErrors = true; mutex.Unlock()
			}
		}(account)
	}
	waitGroup.Wait()
}

func (s *accountServiceImpl) initMetricLensFilters() {
	var (
		waitGroup sync.WaitGroup
		mutex  sync.RWMutex
	)
	for _, account := range s.accounts {
		account.metricLensFilters = map[string]string{}
		for filterID, filter := range account.filters {
			waitGroup.Add(1)
			go func(account *Account, filterID string, filter string) {
				ctx, cancel := context.WithTimeout(s.ctx, *s.timeout)
				defer waitGroup.Done()
				defer cancel()
				for _, dimID := range account.metricLensDimensions   {
					if err := (*s.httpClient).Get(ctx, &map[string]interface{}{}, fmt.Sprintf(metricLensURLFormat, "quality_metriclens", account.id, filterID, int(dimID))); err == nil {
						mutex.Lock(); account.metricLensFilters[filterID] = filter; mutex.Unlock()
					}
					break
				}
			}(account, filterID, filter)
		}
	}
	waitGroup.Wait()
}
