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
			"correlation": r.Correlation,
		}).Info("Added correlation")
	}
}

func (cc *Client) Correlate(cor *Correlation) {
	err := cc.putRequestOnChan(&request{Correlation: cor, operation: http.MethodPut, callback: cc.correlateCb})
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"correlation": cor,
		}).Error("Unable to correlate dimension, not retrying")
	}
}

func (cc *Client) deleteCb(r *request, _ []byte, _ error) {
	if cc.logUpdates {
		log.WithFields(log.Fields{
			"correlation": r.Correlation,
		}).Info("Deleted correlation")
	}
}

func (cc *Client) Delete(cor *Correlation) {
	err := cc.putRequestOnChan(&request{Correlation: cor, operation: http.MethodDelete, callback: cc.deleteCb})
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"correlation": cor,
		}).Error("Unable to delete correlation on dimension, not retrying")
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
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"dimensionName":  dimName,
			"dimensionValue": dimValue,
		}).Error("Unable to get correlations for dimension, not retrying")
	}
}

func (cc *Client) makeRequest(r *request) error {
	var (
		req *http.Request
		err error
	)

	if r.DimName == "" || r.DimValue == "" {
		atomic.AddInt64(&cc.TotalInvalidDimensions, int64(1))
		return errors.New("correlation dimension is missing key or value, cannot send")
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
		return fmt.Errorf("unknown operation for correlation client")
	}

	if err != nil {
		return err
	}

	req.Header.Add("X-SF-TOKEN", cc.Token)

	req = req.WithContext(
		context.WithValue(req.Context(), requests.RequestFailedCallbackKey, requests.RequestFailedCallback(func(statusCode int, err error) {
			if statusCode >= 400 && statusCode < 500 {
				atomic.AddInt64(&cc.TotalClientError4xxResponses, int64(1))

				// don't log a message if we get 404 NotFound on GET
				if statusCode == 404 && r.operation == http.MethodGet {
					// don't retry on 4xx errors
					return
				}

				log.WithError(err).WithFields(log.Fields{
					"url":         req.URL.String(),
					"correlation": r,
				}).Error("Unable to update dimension, not retrying")

				// Don't retry if it is a 4xx error since these
				// imply an input/auth error, which is not going to be remedied
				// by retrying.
				return
			}

			// The retry is meant to provide some measure of robustness against
			// temporary API failures.  If the API is down for significant
			// periods of time, correlation updates will probably eventually back
			// up beyond conf.PropertiesMaxBuffered and start dropping.
			log.WithError(err).WithFields(log.Fields{
				"url":         req.URL.String(),
				"correlation": r,
			}).Error("Unable to update dimension, retrying")
			select {
			// on non 400 errors push the request back on the requestChan to retry,
			// but drop the requests if the requestChan is full.
			// This is not ideal because we won't resend the request,
			// but this is a necessary safety mechanism to prevent the agent from
			// locking up.
			case <-cc.ctx.Done():
				return
			case cc.requestChan <- r:
				atomic.AddInt64(&cc.TotalRetriedUpdates, int64(1))
				return
			default:
				log.WithFields(log.Fields{
					"error": err,
					"dim":   r.DimName,
				}).Errorf("Failed to retry dimension update")
			}
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
					"correlation": r,
				}).Error("Unable to update correlation, not retrying")
			}
		}
	}
}

// Start the client's processing queue
func (cc *Client) Start() {
	go cc.processChan()
}
