package forwarder

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/signalfx/gateway/protocol/signalfx"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/golib/web"
)

type pathSetupFunc = func(*mux.Router, http.Handler, string)

func startListening(ctx context.Context, listenAddr string, timeout time.Duration, sink signalfx.Sink) (sfxclient.Collector, error) {
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, errors.WithMessage(err, "cannot open listening address "+listenAddr)
	}
	router := mux.NewRouter()

	httpChain := web.NextConstructor(func(ctx context.Context, rw http.ResponseWriter, r *http.Request, next web.ContextHandler) {
		next.ServeHTTPC(tryToExtractRemoteAddressToContext(ctx, r), rw, r)
	})

	jaegerMetrics := setupHandler(ctx, router, signalfx.JaegerV1, sink, func(sink signalfx.Sink) signalfx.ErrorReader {
		return signalfx.NewJaegerThriftTraceDecoderV1(golibLogger, sink)
	}, httpChain, signalfx.SetupThriftByPaths, signalfx.DefaultTracePathV1)

	protobufDatapoints := setupHandler(ctx, router, "protobufv2", sink, func(sink signalfx.Sink) signalfx.ErrorReader {
		return &signalfx.ProtobufDecoderV2{Sink: sink, Logger: golibLogger}
	}, httpChain, signalfx.SetupProtobufV2ByPaths, "/v2/datapoint")

	jsonDatapoints := setupHandler(ctx, router, "jsonv2", sink, func(sink signalfx.Sink) signalfx.ErrorReader {
		return &signalfx.JSONDecoderV2{Sink: sink, Logger: golibLogger}
	}, httpChain, signalfx.SetupJSONByPaths, "/v2/datapoint")

	zipkinMetrics := setupHandler(ctx, router, signalfx.ZipkinV1, sink, func(sink signalfx.Sink) signalfx.ErrorReader {
		return &signalfx.JSONTraceDecoderV1{Logger: golibLogger, Sink: sink}
	}, httpChain, signalfx.SetupJSONByPaths, signalfx.DefaultTracePathV1)

	router.NotFoundHandler = http.HandlerFunc(notFoundHandler)

	server := http.Server{
		Handler:      router,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	}

	go func() { _ = server.Serve(listener) }()

	go func() {
		<-ctx.Done()
		err := server.Close()
		if err != nil {
			logger.WithError(err).Error("Could not close SignalFx forwarding server")
		}
	}()
	return sfxclient.NewMultiCollector(jsonDatapoints, protobufDatapoints, jaegerMetrics, zipkinMetrics), nil
}

func setupHandler(ctx context.Context, router *mux.Router, chainType string, sink signalfx.Sink, getReader func(signalfx.Sink) signalfx.ErrorReader, httpChain web.NextConstructor, pathSetup pathSetupFunc, path string) sfxclient.Collector {
	handler, internalMetrics := signalfx.SetupChain(ctx, sink, chainType, getReader, httpChain, golibLogger)
	pathSetup(router, handler, path)
	return internalMetrics
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	errMsg := fmt.Sprintf("Datapoint or span request received on invalid path '%s'. "+
		"You should send to the same path that you would on the Smart Gateway.", r.URL.Path)
	logger.ThrottledError(errMsg)
	_, _ = w.Write([]byte(errMsg + "\n"))
}
