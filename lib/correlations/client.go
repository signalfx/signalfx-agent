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

	"github.com/signalfx/golib/v3/datapoint"
	"go.uber.org/zap"

	"github.com/signalfx/signalfx-agent/lib/requests"
	"github.com/signalfx/signalfx-agent/lib/requests/requestcounter"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
)

var ErrChFull = errors.New("request channel full")
var errRetryChFull = errors.New("retry channel full")
var errMaxAttempts = errors.New("maximum attempts exceeded")
var errRequestCancelled = errors.New("request cancelled")

// ErrMaxEntries is an error returned when the correlation endpoint returns a 418 http status
// code indicating that the set of services or environments is too large to add another value
type ErrMaxEntries struct {
	MaxEntries int64 `json:"max,omitempty"`
}

func (m *ErrMaxEntries) Error() string {
	return fmt.Sprintf("max entries %d", m.MaxEntries)
}

var _ error = (*ErrMaxEntries)(nil)

// CorrelationClient is an interface for correlations.Client
type CorrelationClient interface {
	Correlate(*Correlation, CorrelateCB)
	Delete(*Correlation, SuccessfulDeleteCB)
	Get(dimName string, dimValue string, cb SuccessfulGetCB)
	InternalMetrics() []*datapoint.Datapoint
	Start()
}

type request struct {
	*Correlation
	ctx       context.Context
	cancel    context.CancelFunc
	operation string
	callback  func(body []byte, statuscode int, err error)
	sendAt    time.Time
}

// Client is a client for making dimensional correlations
type Client struct {
	sync.RWMutex
	log           *zap.Logger
	ctx           context.Context
	wg            sync.WaitGroup
	Token         string
	APIURL        *url.URL
	client        *http.Client
	requestSender *requests.ReqSender
	requestChan   chan *request
	retryChan     chan *request
	dedup         *deduplicator

	// For easier unit testing
	now        func() time.Time
	logUpdates bool

	sendDelay   time.Duration
	maxAttempts uint32

	TotalClientError4xxResponses int64
	TotalRetriedUpdates          int64
	TotalInvalidDimensions       int64
	dedupPurgeInterval           time.Duration
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
	sender := requests.NewReqSender(ctx, client, conf.PropertiesMaxRequests, "correlation")
	return &Client{
		// TODO
		log:                zap.NewNop(),
		ctx:                ctx,
		Token:              conf.SignalFxAccessToken,
		APIURL:             conf.ParsedAPIURL(),
		requestSender:      sender,
		client:             client,
		now:                time.Now,
		logUpdates:         conf.LogDimensionUpdates,
		requestChan:        make(chan *request, conf.PropertiesMaxBuffered),
		retryChan:          make(chan *request, conf.PropertiesMaxBuffered),
		dedup:              newDeduplicator(int(conf.PropertiesMaxBuffered)),
		sendDelay:          time.Duration(conf.PropertiesSendDelaySeconds) * time.Second,
		maxAttempts:        uint32(conf.TraceHostCorrelationMaxRequestRetries) + 1,
		dedupPurgeInterval: conf.TraceHostCorrelationPurgeInterval.AsDuration(),
	}, nil
}

func (cc *Client) putRequestOnChan(r *request) error {
	// prevent requests against empty dimension names and values
	if r.DimName == "" || r.DimValue == "" {
		// logging this as debug because this means there's no actual dimension to correlate with
		// and because this isn't being taken off on the request sender and subject to retries, this could
		// potentially spam the logs
		atomic.AddInt64(&cc.TotalInvalidDimensions, int64(1))
		cc.log.Debug("No dimension key or value to correlate to", zap.String("method", r.operation), zap.Any("correlation", r.Correlation))
		return nil
	}

	r.ctx, r.cancel = context.WithCancel(requestcounter.ContextWithRequestCounter(context.Background()))

	var err error
	select {
	case cc.requestChan <- r:
	case <-cc.ctx.Done():
		err = context.DeadlineExceeded
	default:
		err = ErrChFull
	}
	return err
}

func (cc *Client) putRequestOnRetryChan(r *request) error {
	// handle request counter
	if requestcounter.GetRequestCount(r.ctx) == cc.maxAttempts {
		return errMaxAttempts
	}
	requestcounter.IncrementRequestCount(r.ctx)

	// set the time to retry
	r.sendAt = cc.now().Add(cc.sendDelay)

	if r.ctx.Err() != nil {
		return errRequestCancelled
	}

	var err error
	select {
	case <-r.ctx.Done():
		err = errRequestCancelled
	case cc.retryChan <- r:
	case <-cc.ctx.Done():
		err = context.DeadlineExceeded
	default:
		err = errRetryChFull
	}

	return err
}

// CorrelateCB is a call back invoked with Correlate requests
// it is not invoked if the reqeust is deduplicated, cancelled, or the client context is cancelled
type CorrelateCB func(cor *Correlation, err error)

// Correlate
func (cc *Client) Correlate(cor *Correlation, cb CorrelateCB) {
	err := cc.putRequestOnChan(&request{
		Correlation: cor,
		operation:   http.MethodPut,
		callback: func(body []byte, statuscode int, err error) {
			switch statuscode {
			case http.StatusOK:
				if cc.logUpdates {
					cc.log.Info("Updated dimension", zap.String("method", http.MethodPut), zap.Any("correlation", cor))
				}
			case http.StatusTeapot:
				max := &ErrMaxEntries{}
				err = json.Unmarshal(body, max)
				if err == nil {
					err = max
				}
			}
			if err != nil {
				cc.log.Error("Unable to update dimension, not retrying", zap.Error(err), zap.String("method", http.MethodPut), zap.Any("correlation", cor))
			}
			cb(cor, err)
		}})
	if err != nil {
		cc.log.Debug("Unable to update dimension, not retrying", zap.Error(err), zap.String("method", http.MethodPut), zap.Any("correlation", cor))
	}
}

// SuccessfulDeleteCB is a call back that is only invoked on successful Deletion operations
type SuccessfulDeleteCB func(cor *Correlation)

// Delete removes a correlation
func (cc *Client) Delete(cor *Correlation, callback SuccessfulDeleteCB) {
	err := cc.putRequestOnChan(&request{
		Correlation: cor,
		operation:   http.MethodDelete,
		callback: func(_ []byte, statuscode int, err error) {
			switch statuscode {
			case http.StatusOK:
				callback(cor)
				if cc.logUpdates {
					cc.log.Info("Updated dimension", zap.String("method", http.MethodDelete), zap.Any("correlation", cor))
				}
			default:
				cc.log.Error("Unable to update dimension, not retrying", zap.Error(err))
			}
		}})
	if err != nil {
		cc.log.Debug("Unable to update dimension, not retrying", zap.Error(err), zap.String("method", http.MethodDelete), zap.Any("correlation", cor))
	}
}

// SuccessfulGetCB
type SuccessfulGetCB func(map[string][]string)

// Get
func (cc *Client) Get(dimName string, dimValue string, callback SuccessfulGetCB) {
	err := cc.putRequestOnChan(&request{
		Correlation: &Correlation{
			DimName:  dimName,
			DimValue: dimValue,
		},
		operation: http.MethodGet,
		callback: func(body []byte, statuscode int, err error) {
			switch statuscode {
			case http.StatusOK:
				var response = map[string][]string{}
				err = json.Unmarshal(body, &response)
				if err != nil {
					cc.log.Error("Unable to unmarshall correlations for dimension", zap.Error(err), zap.String("dim", dimName), zap.String("value", dimValue))
					return
				}
				callback(response)
			case http.StatusNotFound:
				// only log this as debug because we do a blanket fetch of correlations on the backend
				// and if the backend fails to find anything this isn't really an error for us
				cc.log.Debug("Unable to update dimension, not retrying", zap.Error(err))
			default:
				cc.log.Error("Unable to update dimension, not retrying", zap.Error(err))
			}
		},
	})
	if err != nil {
		cc.log.Debug("Unable to retrieve correlations for dimension, not retrying", zap.Error(err), zap.String("dimensionName", dimName), zap.String("dimensionValue", dimValue))
	}
}

func (cc *Client) makeRequest(r *request) {
	var (
		req *http.Request
		err error
	)

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
		err = fmt.Errorf("unknown operation")
	}

	if err != nil {
		// logging this as debug because this means there's something fundamentally wrong with the request
		// and because this isn't being taken off on the request sender and subject to retries, this could
		// potentially spam the logs long term.  This would be a really good candidate for a throttled error logger
		cc.log.Debug("Unable to make request, not retrying", zap.Error(err), zap.String("method", r.operation), zap.Any("correlation", r.Correlation))
		r.cancel()
		return
	}

	req.Header.Add("X-SF-TOKEN", cc.Token)

	req = req.WithContext(
		context.WithValue(req.Context(), requests.RequestFailedCallbackKey, requests.RequestFailedCallback(func(body []byte, statusCode int, err error) {
			// retry if the http status code is not 4XX. A 4xx or http client error implies
			// an error that is not going to be remedied by retrying.
			if statusCode < 400 || statusCode >= 500 {
				// The retry (for non 400 errors) is meant to provide some measure of robustness against
				// temporary API failures.  If the API is down for significant
				// periods of time, correlation updates will probably eventually back
				// up beyond conf.PropertiesMaxBuffered and start dropping.
				retryErr := cc.putRequestOnRetryChan(r)
				if retryErr == nil {
					cc.log.Debug("Unable to update dimension, retrying", zap.Error(err), zap.String("method", req.Method), zap.Any("correlation", r.Correlation))
					return
				}
			} else {
				atomic.AddInt64(&cc.TotalClientError4xxResponses, int64(1))
			}

			// invoke the callback
			r.callback(body, statusCode, err)

			// cancel the request context
			r.cancel()
		})))

	req = req.WithContext(
		context.WithValue(req.Context(), requests.RequestSuccessCallbackKey, requests.RequestSuccessCallback(func(body []byte) {
			r.callback(body, http.StatusOK, nil)
			// close the request context
			r.cancel()
		})))

	// This will block if we don't have enough requests
	cc.requestSender.Send(req)
}

// routines
// processChan processes incoming requests, drops duplicates, and cancels conflicting requests
func (cc *Client) processChan() {
	defer cc.wg.Done()
	purgeDeduper := time.NewTimer(cc.dedupPurgeInterval)
	defer purgeDeduper.Stop()
	for {
		select {
		case <-cc.ctx.Done():
			return
		case <-purgeDeduper.C:
			cc.dedup.purge()
			purgeDeduper.Reset(cc.dedupPurgeInterval)
		case r := <-cc.requestChan:
			if cc.dedup.isDup(r) {
				r.cancel()
				continue
			}
			cc.makeRequest(r)
		}
	}
}

// processRetryChan is a routine that drains the retry channel and waits until the appropriate time to retry the request
func (cc *Client) processRetryChan() {
	defer cc.wg.Done()
	for {
		select {
		case <-cc.ctx.Done(): // client is shutdown
			return
		case r := <-cc.retryChan:
			if r.ctx.Err() != nil {
				continue
			}
			select {
			case <-time.After(time.Until(r.sendAt)): // wait and resend the request
				atomic.AddInt64(&cc.TotalRetriedUpdates, int64(1))
				cc.makeRequest(r)
			case <-r.ctx.Done(): // request is cancelled
				continue
			case <-cc.ctx.Done(): // client is shutdown
				return
			}
		}
	}
}

// Start the client's processing queue
func (cc *Client) Start() {
	cc.wg.Add(2)
	go cc.processChan()
	go cc.processRetryChan()
}
