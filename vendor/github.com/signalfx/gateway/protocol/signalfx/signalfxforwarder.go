package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/signalfx/gateway/protocol/filtering"
	"github.com/signalfx/gateway/sampling"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/datapoint/dpsink"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/golib/log"
	"github.com/signalfx/golib/pointer"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/golib/trace"
	"net"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// Forwarder controls forwarding datapoints to SignalFx
type Forwarder struct {
	filtering.FilteredForwarder
	defaultAuthToken      string
	tr                    *http.Transport
	client                *http.Client
	userAgent             string
	emptyMetricNameFilter dpsink.EmptyMetricFilter

	sink Sink

	jsonMarshal func(v interface{}) ([]byte, error)
	Logger      log.Logger
	stats       stats
	sampler     *sampling.SmartSampler
}

type stats struct {
	totalDatapointsForwarded int64
	totalEventsForwarded     int64
	requests                 *sfxclient.RollingBucket
	drainSize                *sfxclient.RollingBucket
	totalSpansForwarded      int64
	pipeline                 int64
}

// ForwarderConfig controls optional parameters for a signalfx forwarder
type ForwarderConfig struct {
	Filters            *filtering.FilterObj
	DatapointURL       *string
	EventURL           *string
	TraceURL           *string
	Timeout            *time.Duration
	SourceDimensions   *string
	GatewayVersion     *string
	MaxIdleConns       *int64
	AuthToken          *string
	ProtoMarshal       func(pb proto.Message) ([]byte, error)
	JSONMarshal        func(v interface{}) ([]byte, error)
	Logger             log.Logger
	DisableCompression *bool
	TraceSample        *sampling.SmartSampleConfig
}

var defaultForwarderConfig = &ForwarderConfig{
	Filters:            &filtering.FilterObj{},
	DatapointURL:       pointer.String("https://ingest.signalfx.com/v2/datapoint"),
	EventURL:           pointer.String("https://ingest.signalfx.com/v2/event"),
	TraceURL:           pointer.String("https://ingest.signalfx.com/v1/trace"),
	AuthToken:          pointer.String(""),
	Timeout:            pointer.Duration(time.Second * 30),
	GatewayVersion:     pointer.String("UNKNOWN_VERSION"),
	MaxIdleConns:       pointer.Int64(20),
	JSONMarshal:        json.Marshal,
	Logger:             log.Discard,
	DisableCompression: pointer.Bool(false),
}

// NewForwarder creates a new JSON forwarder
func NewForwarder(conf *ForwarderConfig) (ret *Forwarder, err error) {
	conf = pointer.FillDefaultFrom(conf, defaultForwarderConfig).(*ForwarderConfig)
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConnsPerHost:   int(*conf.MaxIdleConns * 2),
		ResponseHeaderTimeout: *conf.Timeout,
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, *conf.Timeout)
		},
		TLSHandshakeTimeout: *conf.Timeout,
	}
	sendingSink := sfxclient.NewHTTPSink()
	sendingSink.DisableCompression = *conf.DisableCompression
	sendingSink.Client = &http.Client{
		Transport: tr,
		Timeout:   *conf.Timeout,
	}
	sendingSink.AuthToken = *conf.AuthToken
	sendingSink.UserAgent = fmt.Sprintf("SignalfxGateway/%s (gover %s)", *conf.GatewayVersion, runtime.Version())
	sendingSink.DatapointEndpoint = *conf.DatapointURL
	sendingSink.EventEndpoint = *conf.EventURL
	sendingSink.TraceEndpoint = *conf.TraceURL
	ret = &Forwarder{
		defaultAuthToken: sendingSink.AuthToken,
		userAgent:        sendingSink.UserAgent,
		tr:               tr,
		client:           sendingSink.Client,
		jsonMarshal:      conf.JSONMarshal,
		sink:             sendingSink,
		Logger:           conf.Logger,
		stats: stats{
			requests: sfxclient.NewRollingBucket("request_time.ns", map[string]string{
				"direction":   "forwarder",
				"destination": "signalfx",
			}),
			drainSize: sfxclient.NewRollingBucket("drain_size", map[string]string{
				"direction":   "forwarder",
				"destination": "signalfx",
			}),
		},
	}
	err = ret.Setup(conf.Filters)
	if err == nil {
		if conf.TraceSample != nil {
			ret.sampler, err = sampling.New(conf.TraceSample, conf.Logger, sendingSink)
			ret.sampler.ConfigureHTTPSink(sendingSink)
		}
		return ret, err
	}
	return nil, err
}

// Datapoints returns datapoints
func (connector *Forwarder) Datapoints() []*datapoint.Datapoint {
	dps := connector.stats.requests.Datapoints()
	dps = append(dps, connector.stats.drainSize.Datapoints()...)
	dps = append(dps, connector.GetFilteredDatapoints()...)
	dps = append(dps, connector.sampler.Datapoints()...)
	return dps
}

// Close will terminate idle HTTP client connections
func (connector *Forwarder) Close() error {
	connector.tr.CloseIdleConnections()
	return connector.sampler.Close()
}

// TokenHeaderName is the header key for the auth token in the HTTP request
const TokenHeaderName = "X-SF-TOKEN"

// AddDatapoints forwards datapoints to SignalFx
func (connector *Forwarder) AddDatapoints(ctx context.Context, datapoints []*datapoint.Datapoint) error {
	start := time.Now()
	atomic.AddInt64(&connector.stats.pipeline, int64(len(datapoints)))
	defer atomic.AddInt64(&connector.stats.pipeline, -int64(len(datapoints)))
	defer connector.stats.requests.Add(float64(time.Now().Sub(start).Nanoseconds()))
	defer connector.stats.drainSize.Add(float64(len(datapoints)))
	atomic.AddInt64(&connector.stats.totalDatapointsForwarded, int64(len(datapoints)))
	datapoints = connector.emptyMetricNameFilter.FilterDatapoints(datapoints)
	datapoints = connector.FilterDatapoints(datapoints)
	if len(datapoints) == 0 {
		return nil
	}
	return connector.sink.AddDatapoints(ctx, datapoints)
}

// AddEvents forwards events to SignalFx
func (connector *Forwarder) AddEvents(ctx context.Context, events []*event.Event) error {
	atomic.AddInt64(&connector.stats.pipeline, int64(len(events)))
	defer atomic.AddInt64(&connector.stats.pipeline, -int64(len(events)))
	atomic.AddInt64(&connector.stats.totalEventsForwarded, int64(len(events)))
	// could filter here
	if len(events) == 0 {
		return nil
	}
	return connector.sink.AddEvents(ctx, events)
}

// AddSpans forwards traces to SignalFx, optionally through the sampler
func (connector *Forwarder) AddSpans(ctx context.Context, spans []*trace.Span) error {
	atomic.AddInt64(&connector.stats.pipeline, int64(len(spans)))
	defer atomic.AddInt64(&connector.stats.pipeline, -int64(len(spans)))
	atomic.AddInt64(&connector.stats.totalSpansForwarded, int64(len(spans)))
	// could filter here
	if len(spans) == 0 {
		return nil
	}
	return connector.sampler.AddSpans(ctx, spans, connector.sink)
}

// Pipeline returns the total of all things forwarded
func (connector *Forwarder) Pipeline() int64 {
	return atomic.LoadInt64(&connector.stats.pipeline)
}

// StartupFinished calls the same interface on the sampler as a hook called by run() after the gateway is up and running
func (connector *Forwarder) StartupFinished() error {
	return connector.sampler.StartupFinished()
}
