package signalfx

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"bytes"
	"context"

	"github.com/gorilla/mux"
	"github.com/signalfx/com_signalfx_metrics_protobuf"
	"github.com/signalfx/gateway/logkey"
	"github.com/signalfx/gateway/protocol"
	"github.com/signalfx/gateway/protocol/collectd"
	"github.com/signalfx/gateway/protocol/signalfx/tagreplace"
	"github.com/signalfx/gateway/protocol/zipper"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/datapoint/dpsink"
	"github.com/signalfx/golib/errors"
	"github.com/signalfx/golib/log"
	"github.com/signalfx/golib/pointer"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/golib/web"
)

// ListenerServer controls listening on a socket for SignalFx connections
type ListenerServer struct {
	protocol.CloseableHealthCheck
	listener net.Listener
	logger   log.Logger

	internalCollectors sfxclient.Collector
	metricHandler      metricHandler
}

// Close the exposed socket listening for new connections
func (streamer *ListenerServer) Close() error {
	return streamer.listener.Close()
}

// Addr returns the currently listening address
func (streamer *ListenerServer) Addr() net.Addr {
	return streamer.listener.Addr()
}

// Datapoints returns the datapoints about various internal endpoints
func (streamer *ListenerServer) Datapoints() []*datapoint.Datapoint {
	return append(streamer.internalCollectors.Datapoints(), streamer.HealthDatapoints()...)
}

// MericTypeGetter is an old metric interface that returns the type of a metric name
type MericTypeGetter interface {
	GetMetricTypeFromMap(metricName string) com_signalfx_metrics_protobuf.MetricType
}

// ErrorReader are datapoint streamers that read from a HTTP request and return errors if
// the stream is invalid
type ErrorReader interface {
	Read(ctx context.Context, req *http.Request) error
}

// ErrorTrackerHandler behaves like a http handler, but tracks error returns from a ErrorReader
type ErrorTrackerHandler struct {
	TotalErrors int64
	reader      ErrorReader
	Logger      log.Logger
}

// Datapoints gets TotalErrors stats
func (e *ErrorTrackerHandler) Datapoints() []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		sfxclient.Cumulative("total_errors", nil, atomic.LoadInt64(&e.TotalErrors)),
	}
}

// ServeHTTPC will serve the wrapped ErrorReader and return the error (if any) to rw if ErrorReader
// fails
func (e *ErrorTrackerHandler) ServeHTTPC(ctx context.Context, rw http.ResponseWriter, req *http.Request) {
	if err := e.reader.Read(ctx, req); err != nil {
		atomic.AddInt64(&e.TotalErrors, 1)
		rw.WriteHeader(http.StatusBadRequest)
		_, err = rw.Write([]byte(err.Error()))
		log.IfErr(e.Logger, err)
		return
	}
	_, err := rw.Write([]byte(`"OK"`))
	log.IfErr(e.Logger, err)
}

// ListenerConfig controls optional parameters for the listener
type ListenerConfig struct {
	ListenAddr               *string
	HealthCheck              *string
	Timeout                  *time.Duration
	Logger                   log.Logger
	RootContext              context.Context
	JSONMarshal              func(v interface{}) ([]byte, error)
	DebugContext             *web.HeaderCtxFlag
	HTTPChain                web.NextConstructor
	SpanNameReplacementRules []string
}

var defaultListenerConfig = &ListenerConfig{
	ListenAddr:               pointer.String("127.0.0.1:12345"),
	HealthCheck:              pointer.String("/healthz"),
	Timeout:                  pointer.Duration(time.Second * 30),
	Logger:                   log.Discard,
	RootContext:              context.Background(),
	JSONMarshal:              json.Marshal,
	SpanNameReplacementRules: []string{},
}

type metricHandler struct {
	metricCreationsMapMutex sync.Mutex
	metricCreationsMap      map[string]com_signalfx_metrics_protobuf.MetricType
	jsonMarshal             func(v interface{}) ([]byte, error)
	logger                  log.Logger
}

func (handler *metricHandler) ServeHTTP(writter http.ResponseWriter, req *http.Request) {
	dec := json.NewDecoder(req.Body)
	var d []MetricCreationStruct
	if err := dec.Decode(&d); err != nil {
		handler.logger.Log(log.Err, err, "Invalid metric creation request")
		writter.WriteHeader(http.StatusBadRequest)
		_, err = writter.Write([]byte(`{msg:"Invalid creation request"}`))
		log.IfErr(handler.logger, err)
		return
	}
	handler.metricCreationsMapMutex.Lock()
	defer handler.metricCreationsMapMutex.Unlock()
	ret := []MetricCreationResponse{}
	for _, m := range d {
		metricType, ok := com_signalfx_metrics_protobuf.MetricType_value[m.MetricType]
		if !ok {
			writter.WriteHeader(http.StatusBadRequest)
			_, err := writter.Write([]byte(`{msg:"Invalid metric type"}`))
			log.IfErr(handler.logger, err)
			return
		}
		handler.metricCreationsMap[m.MetricName] = com_signalfx_metrics_protobuf.MetricType(metricType)
		ret = append(ret, MetricCreationResponse{Code: 409})
	}
	toWrite, err := handler.jsonMarshal(ret)
	if err != nil {
		handler.logger.Log(log.Err, err, "Unable to marshal json")
		writter.WriteHeader(http.StatusBadRequest)
		_, err = writter.Write([]byte(`{msg:"Unable to marshal json!"}`))
		log.IfErr(handler.logger, err)
		return
	}
	writter.WriteHeader(http.StatusOK)
	_, err = writter.Write(toWrite)
	log.IfErr(handler.logger, err)
}

func (handler *metricHandler) GetMetricTypeFromMap(metricName string) com_signalfx_metrics_protobuf.MetricType {
	handler.metricCreationsMapMutex.Lock()
	defer handler.metricCreationsMapMutex.Unlock()
	mt, ok := handler.metricCreationsMap[metricName]
	if !ok {
		return com_signalfx_metrics_protobuf.MetricType_GAUGE
	}
	return mt
}

// NewListener servers http requests for Signalfx datapoints
func NewListener(sink Sink, conf *ListenerConfig) (*ListenerServer, error) {
	conf = pointer.FillDefaultFrom(conf, defaultListenerConfig).(*ListenerConfig)

	listener, err := net.Listen("tcp", *conf.ListenAddr)
	if err != nil {
		return nil, errors.Annotatef(err, "cannot open listening address %s", *conf.ListenAddr)
	}
	r := mux.NewRouter()

	server := http.Server{
		Handler:      r,
		Addr:         *conf.ListenAddr,
		ReadTimeout:  *conf.Timeout,
		WriteTimeout: *conf.Timeout,
	}
	listenServer := ListenerServer{
		listener: listener,
		logger:   conf.Logger,
		metricHandler: metricHandler{
			metricCreationsMap: make(map[string]com_signalfx_metrics_protobuf.MetricType),
			logger:             log.NewContext(conf.Logger).With(logkey.Struct, "metricHandler"),
			jsonMarshal:        conf.JSONMarshal,
		},
	}
	listenServer.SetupHealthCheck(conf.HealthCheck, r, conf.Logger)

	r.Handle("/v1/metric", &listenServer.metricHandler)
	r.Handle("/metric", &listenServer.metricHandler)

	traceSink := sink
	if len(conf.SpanNameReplacementRules) > 0 {
		var err error
		traceSink, err = tagreplace.New(conf.SpanNameReplacementRules, sink)
		if err != nil {
			return nil, errors.Annotatef(err, "cannot parse tag replacement rules %v", conf.SpanNameReplacementRules)
		}
	}

	listenServer.internalCollectors = sfxclient.NewMultiCollector(
		setupNotFoundHandler(conf.RootContext, r),
		setupProtobufV1(conf.RootContext, r, sink, &listenServer.metricHandler, conf.Logger, conf.HTTPChain),
		setupJSONV1(conf.RootContext, r, sink, &listenServer.metricHandler, conf.Logger, conf.HTTPChain),
		setupProtobufV2(conf.RootContext, r, sink, conf.Logger, conf.DebugContext, conf.HTTPChain),
		setupProtobufEventV2(conf.RootContext, r, sink, conf.Logger, conf.DebugContext, conf.HTTPChain),
		setupJSONV2(conf.RootContext, r, sink, conf.Logger, conf.DebugContext, conf.HTTPChain),
		setupJSONEventV2(conf.RootContext, r, sink, conf.Logger, conf.DebugContext, conf.HTTPChain),
		setupCollectd(conf.RootContext, r, sink, conf.DebugContext, conf.HTTPChain, conf.Logger),
		setupThriftTraceV1(conf.RootContext, r, traceSink, conf.Logger, conf.HTTPChain),
		setupJSONTraceV1(conf.RootContext, r, traceSink, conf.Logger, conf.HTTPChain),
	)

	go func() {
		log.IfErr(conf.Logger, server.Serve(listener))
	}()
	return &listenServer, err
}

func setupNotFoundHandler(ctx context.Context, r *mux.Router) sfxclient.Collector {
	metricTracking := web.RequestCounter{}
	r.NotFoundHandler = web.NewHandler(ctx, web.FromHTTP(http.NotFoundHandler())).Add(web.NextHTTP(metricTracking.ServeHTTP))
	return &sfxclient.WithDimensions{
		Dimensions: map[string]string{"http_endpoint": "http404"},
		Collector:  &metricTracking,
	}
}

// SetupChain wraps the reader returned by getReader in an http.Handler along
// with some middleware that calculates internal metrics about requests.
func SetupChain(ctx context.Context, sink Sink, chainType string, getReader func(Sink) ErrorReader, httpChain web.NextConstructor, logger log.Logger, moreConstructors ...web.Constructor) (http.Handler, sfxclient.Collector) {
	zippers := zipper.NewZipper()

	counter := &dpsink.Counter{
		Logger: logger,
	}

	ucount := UnifyNextSinkWrap(counter)
	finalSink := FromChain(sink, NextWrap(ucount))
	errReader := getReader(finalSink)
	errorTracker := ErrorTrackerHandler{
		reader: errReader,
		Logger: logger,
	}
	metricTracking := web.RequestCounter{}
	handler := web.NewHandler(ctx, &errorTracker).Add(web.NextHTTP(metricTracking.ServeHTTP)).Add(httpChain)
	for _, c := range moreConstructors {
		handler.Add(c)
	}
	st := &sfxclient.WithDimensions{
		Collector: sfxclient.NewMultiCollector(
			&metricTracking,
			&errorTracker,
			counter,
			zippers,
		),
		Dimensions: map[string]string{
			"http_endpoint": "sfx_" + chainType,
		},
	}
	return zippers.GzipHandler(handler), st
}

// SetupJSONByPaths tells the router which paths the given handler (which should handle the given
// endpoint) should see
func SetupJSONByPaths(r *mux.Router, handler http.Handler, endpoint string) {
	r.Path(endpoint).Methods("POST").Headers("Content-Type", "application/json").Handler(handler)
	r.Path(endpoint).Methods("POST").Headers("Content-Type", "application/json; charset=UTF-8").Handler(handler)
	r.Path(endpoint).Methods("POST").Headers("Content-Type", "").HandlerFunc(web.InvalidContentType)
	r.Path(endpoint).Methods("POST").Handler(handler)
}

func setupCollectd(ctx context.Context, r *mux.Router, sink dpsink.Sink, debugContext *web.HeaderCtxFlag, httpChain web.NextConstructor, logger log.Logger) sfxclient.Collector {
	counter := &dpsink.Counter{
		Logger: logger,
	}
	finalSink := dpsink.FromChain(sink, dpsink.NextWrap(counter))
	decoder := collectd.JSONDecoder{
		Logger: logger,
		SendTo: finalSink,
	}
	metricTracking := &web.RequestCounter{}
	httpHandler := web.NewHandler(ctx, &decoder).Add(web.NextHTTP(metricTracking.ServeHTTP), debugContext, httpChain)
	collectd.SetupCollectdPaths(r, httpHandler, "/v1/collectd")
	return &sfxclient.WithDimensions{
		Collector: sfxclient.NewMultiCollector(
			metricTracking,
			counter,
			&decoder,
		),
		Dimensions: map[string]string{
			"type": "collectd",
		},
	}
}

func readFromRequest(jeff *bytes.Buffer, req *http.Request, logger log.Logger) error {
	// for compressed transactions, contentLength isn't trustworthy
	readLen, err := jeff.ReadFrom(req.Body)
	if err != nil {
		logger.Log(log.Err, err, logkey.ReadLen, readLen, logkey.ContentLength, req.ContentLength, "Unable to fully read from buffer")
		return err
	}
	return nil
}
