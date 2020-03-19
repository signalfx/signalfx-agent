package correlations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/core/writer/requests"
	log "github.com/sirupsen/logrus"
)

var errChFull = errors.New("request channel full")

// CorrelationClient is an interface for correlations.Client
type CorrelationClient interface {
	Correlate(correlation *Correlation)
	Delete(correlation *Correlation)
	Get(dimName string, dimValue string, callback func(map[string][]string, error))
	Start()
}

type request struct {
	*Correlation
	operation string
	callback  func(*request, []byte, error)
}

// Client is a client for making dimensional correlations
type Client struct {
	sync.RWMutex
	ctx           context.Context
	Token         string
	APIURL        *url.URL
	client        *http.Client
	requestSender *requests.ReqSender
	requestChan   chan *request
	// For easier unit testing
	now        func() time.Time
	logUpdates bool

	TotalClientError4xxResponses int64
	TotalRetriedUpdates          int64
	TotalInvalidDimensions       int64
}

// NewCorrelationClient returns a new Client
func NewCorrelationClient(ctx context.Context, conf *config.WriterConfig) (CorrelationClient, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:        int(conf.PropertiesMaxRequests),
			MaxIdleConnsPerHost: int(conf.PropertiesMaxRequests),
			IdleConnTimeout:     30 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	sender := requests.NewReqSender(ctx, client, conf.PropertiesMaxRequests, map[string]string{"client": "correlation"})

	return &Client{
		ctx:           ctx,
		Token:         conf.SignalFxAccessToken,
		APIURL:        conf.ParsedAPIURL(),
		requestSender: sender,
		client:        client,
		now:           time.Now,
		logUpdates:    conf.LogDimensionUpdates,
		requestChan:   make(chan *request, conf.PropertiesMaxBuffered),
	}, nil
}

func (cc *Client) putRequestOnChan(r *request) error {
	var err error
	select {
	case cc.requestChan <- r:
	case <-cc.ctx.Done():
		err = context.DeadlineExceeded
	default:
		err = errChFull
	}
	return err
}

func (cc *Client) correlateCb(r *request, _ []byte, _ error) {
	if cc.logUpdates {
		log.WithFields(log.Fields{
			"method":      http.MethodPut,
			"correlation": r.Correlation,
		}).Info("Updated dimension")
	}
}

func (cc *Client) Correlate(cor *Correlation) {
	err := cc.putRequestOnChan(&request{Correlation: cor, operation: http.MethodPut, callback: cc.correlateCb})
	if err != nil && err != context.DeadlineExceeded {
		log.WithError(err).WithFields(log.Fields{
			"method":      http.MethodPut,
			"correlation": cor,
		}).Error("Unable to update dimension, not retrying")
	}
}

func (cc *Client) deleteCb(r *request, _ []byte, _ error) {
	if cc.logUpdates {
		log.WithFields(log.Fields{
			"method":      http.MethodDelete,
			"correlation": r.Correlation,
		}).Info("Updated dimension")
	}
}

func (cc *Client) Delete(cor *Correlation) {
	err := cc.putRequestOnChan(&request{Correlation: cor, operation: http.MethodDelete, callback: cc.deleteCb})
	if err != nil && err != context.DeadlineExceeded {
		log.WithError(err).WithFields(log.Fields{
			"method":      http.MethodDelete,
			"correlation": cor,
		}).Error("Unable to update dimension, not retrying")
	}
}

func (cc *Client) Get(dimName string, dimValue string, callback func(map[string][]string, error)) {
	err := cc.putRequestOnChan(&request{
		Correlation: &Correlation{
			DimName:  dimName,
			DimValue: dimValue,
		},
		operation: http.MethodGet,
		callback: func(r *request, body []byte, _ error) {
			// on success unmarshal the response body and
			// pass it to the call back
			var response = map[string][]string{}
			callback(response, json.Unmarshal(body, &response))
		},
	})
	if err != nil && err != context.DeadlineExceeded {
		log.WithError(err).WithFields(log.Fields{
			"dimensionName":  dimName,
			"dimensionValue": dimValue,
		}).Error("Unable to retrieve correlations for dimension, not retrying")
	}
}

func (cc *Client) makeRequest(r *request) error {
	var (
		req *http.Request
		err error
	)

	if r.DimName == "" || r.DimValue == "" {
		atomic.AddInt64(&cc.TotalInvalidDimensions, int64(1))
		return errors.New("dimension is missing key or value")
	}

	// build endpoint url
	endpoint := fmt.Sprintf("%s/v2/apm/correlate/%s/%s", cc.APIURL, url.PathEscape(r.DimName), url.PathEscape(r.DimValue))

	switch r.operation {
	case http.MethodGet:
		req, err = http.NewRequest(r.operation, endpoint, nil)
	case http.MethodPut:
		// TODO: pool the reader
		endpoint = fmt.Sprintf("%s/%s", endpoint, r.Type)
		req, err = http.NewRequest(r.operation, endpoint, strings.NewReader(r.Value))
		req.Header.Add("Content-Type", "text/plain")
	case http.MethodDelete:
		endpoint = fmt.Sprintf("%s/%s/%s", endpoint, r.Type, url.PathEscape(r.Value))
		req, err = http.NewRequest(r.operation, endpoint, nil)
	default:
		return fmt.Errorf("unknown operation")
	}

	if err != nil {
		return err
	}

	req.Header.Add("X-SF-TOKEN", cc.Token)

	req = req.WithContext(
		context.WithValue(req.Context(), requests.RequestFailedCallbackKey, requests.RequestFailedCallback(func(statusCode int, err error) {
			if statusCode >= 400 && statusCode < 500 {
				// Don't retry if it is a 4xx error since these
				// imply an input/auth error, which is not going to be remedied
				// by retrying.
				atomic.AddInt64(&cc.TotalClientError4xxResponses, int64(1))

				// don't log a message if we get 404 NotFound on GET
				if statusCode == 404 && r.operation == http.MethodGet {
					log.WithError(err).WithFields(log.Fields{
						"method":      req.Method,
						"url":         req.URL.String(),
						"correlation": r.Correlation,
					}).Debug("Unable to update dimension, not retrying")
					return
				}

				log.WithError(err).WithFields(log.Fields{
					"method":      req.Method,
					"url":         req.URL.String(),
					"correlation": r.Correlation,
				}).Error("Unable to update dimension, not retrying")
				return
			}

			// The retry (for non 400 errors) is meant to provide some measure of robustness against
			// temporary API failures.  If the API is down for significant
			// periods of time, correlation updates will probably eventually back
			// up beyond conf.PropertiesMaxBuffered and start dropping.
			retryErr := cc.putRequestOnChan(r)
			if retryErr != nil {
				log.WithError(err).WithFields(log.Fields{
					"method":      req.Method,
					"url":         req.URL.String(),
					"correlation": r.Correlation,
				}).WithError(errChFull).Error("Unable to update dimension, unable to retry")
				return
			}

			// successfully queued request to retry
			atomic.AddInt64(&cc.TotalRetriedUpdates, int64(1))
			log.WithError(err).WithFields(log.Fields{
				"method":      req.Method,
				"url":         req.URL.String(),
				"correlation": r.Correlation,
			}).Error("Unable to update dimension, retrying")
		})))

	req = req.WithContext(
		context.WithValue(req.Context(), requests.RequestSuccessCallbackKey, requests.RequestSuccessCallback(func(body []byte) {
			r.callback(r, body, nil)
		})))

	// This will block if we don't have enough requests
	cc.requestSender.Send(req)

	return nil
}

func (cc *Client) processChan() {
	for {
		select {
		case <-cc.ctx.Done():
			return
		case r := <-cc.requestChan:
			err := cc.makeRequest(r)
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"method":      r.operation,
					"correlation": r.Correlation,
				}).Error("Unable to make request, not retrying")
			}
		}
	}
}

// Start the client's processing queue
func (cc *Client) Start() {
	go cc.processChan()
}
