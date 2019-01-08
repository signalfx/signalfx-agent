package traceforwarder

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/signalfx/gateway/protocol/signalfx"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/golib/trace"
	"github.com/signalfx/golib/web"
)

type pathSetupFunc = func(*mux.Router, http.Handler, string)

func startListeningForSpans(ctx context.Context, listenAddr string, timeout time.Duration, sink trace.Sink) (sfxclient.Collector, error) {
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, errors.WithMessage(err, "cannot open listening address "+listenAddr)
	}
	router := mux.NewRouter()

	httpChain := web.NextConstructor(func(ctx context.Context, rw http.ResponseWriter, r *http.Request, next web.ContextHandler) {
		next.ServeHTTPC(ctx, rw, r)
	})

	jaegerMetrics := setupHandler(ctx, router, signalfx.JaegerV1, &traceOnlySink{sink}, func(sink signalfx.Sink) signalfx.ErrorReader {
		return signalfx.NewJaegerThriftTraceDecoderV1(golibLogger, sink)
	}, httpChain, signalfx.SetupThriftByPaths)

	zipkinMetrics := setupHandler(ctx, router, signalfx.ZipkinV1, &traceOnlySink{sink}, func(sink signalfx.Sink) signalfx.ErrorReader {
		return &signalfx.JSONTraceDecoderV1{Logger: golibLogger, Sink: sink}
	}, httpChain, signalfx.SetupJSONByPaths)

	server := http.Server{
		Handler:      router,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	}

	go server.Serve(listener)

	go func() {
		select {
		case <-ctx.Done():
			err := server.Close()
			if err != nil {
				logger.WithError(err).Error("Could not close trace forwarding server")
			}
		}
	}()
	return sfxclient.NewMultiCollector(jaegerMetrics, zipkinMetrics), nil
}

func setupHandler(ctx context.Context, router *mux.Router, chainType string, sink signalfx.Sink, getReader func(signalfx.Sink) signalfx.ErrorReader, httpChain web.NextConstructor, pathSetup pathSetupFunc) sfxclient.Collector {
	handler, internalMetrics := signalfx.SetupChain(ctx, sink, chainType, getReader, httpChain, golibLogger)
	pathSetup(router, handler, signalfx.DefaultTracePathV1)
	return internalMetrics
}
