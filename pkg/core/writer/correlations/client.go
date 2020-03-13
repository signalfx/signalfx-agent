package correlations

import (
	"context"
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

const correlationChanCapacity = 1000

// CorrelationClient is an interface for correlations.Client
type CorrelationClient interface {
	AcceptCorrelation(cor *Correlation)
	Start()
}

// NOOPClient implements CorrelationClient interface but doesn't do anything with the correlation
type NOOPClient struct{}

func (*NOOPClient) AcceptCorrelation(*Correlation) {}
func (*NOOPClient) Start()                         {}

var _ CorrelationClient = &NOOPClient{}

// Client is a client for making dimensional correlations
type Client struct {
	sync.RWMutex
	ctx             context.Context
	Token           string
	APIURL          *url.URL
	client          *http.Client
	requestSender   *requests.ReqSender
	correlationChan chan *Correlation
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
		ctx:             ctx,
		Token:           conf.SignalFxAccessToken,
		APIURL:          conf.ParsedAPIURL(),
		requestSender:   sender,
		client:          client,
		now:             time.Now,
		logUpdates:      conf.LogDimensionUpdates,
		correlationChan: make(chan *Correlation, correlationChanCapacity),
	}, nil
}

func (cc *Client) AcceptCorrelation(cor *Correlation) {
	cc.correlationChan <- cor
}

func (cc *Client) correlate(cor *Correlation) error {
	var (
		req *http.Request
		err error
	)

	if cor.DimName == "" || cor.DimValue == "" {
		atomic.AddInt64(&cc.TotalInvalidDimensions, int64(1))
		return fmt.Errorf("correlation dimension %v is missing key or value, cannot Send", cor)
	}

	// build endpoint url
	endpoint := fmt.Sprintf("%s/v2/apm/correlate/%s/%s/%s", cc.APIURL, cor.DimName, cor.DimValue, string(cor.Type))

	switch cor.Operation {
	case Put:
		// TODO: pool the reader
		req, err = http.NewRequest(string(cor.Operation), endpoint, strings.NewReader(cor.Value))
		req.Header.Add("Content-Type", "text/plain")
	case Delete:
		endpoint = fmt.Sprintf("%s/%s", endpoint, cor.Value)
		req, err = http.NewRequest(string(cor.Operation), endpoint, nil)
	}

	if err != nil {
		return err
	}

	req.Header.Add("X-SF-TOKEN", cc.Token)

	req = req.WithContext(
		context.WithValue(req.Context(), requests.RequestFailedCallbackKey, requests.RequestFailedCallback(func(statusCode int, err error) {
			if statusCode >= 400 && statusCode < 500 {
				atomic.AddInt64(&cc.TotalClientError4xxResponses, int64(1))
				log.WithError(err).WithFields(log.Fields{
					"url":         req.URL.String(),
					"correlation": cor,
				}).Error("Unable to update dimension, not retrying")

				// Don't retry if it is a 4xx error since these
				// imply an input/auth error, which is not going to be remedied
				// by retrying.
				return
			}

			log.WithError(err).WithFields(log.Fields{
				"url":         req.URL.String(),
				"correlation": cor,
			}).Error("Unable to update dimension, retrying")
			atomic.AddInt64(&cc.TotalRetriedUpdates, int64(1))
			// The retry is meant to provide some measure of robustness against
			// temporary API failures.  If the API is down for significant
			// periods of time, dimension updates will probably eventually back
			// up beyond conf.PropertiesMaxBuffered and start dropping.
			if err := cc.correlate(cor); err != nil {
				log.WithFields(log.Fields{
					"error": err,
					"dim":   cor.DimName,
				}).Errorf("Failed to retry dimension update")
			}
		})))

	req = req.WithContext(
		context.WithValue(req.Context(), requests.RequestSuccessCallbackKey, requests.RequestSuccessCallback(func() {
			if cc.logUpdates {
				log.WithFields(log.Fields{
					"correlation": cor,
				}).Info("Updated dimension")
			}
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
		case correlation := <-cc.correlationChan:
			_ = cc.correlate(correlation)
		}
	}
}

// Start the client's processing queue
func (cc *Client) Start() {
	go cc.processChan()
}
