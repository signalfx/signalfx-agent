package traceforwarder

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/signalfx/golib/trace"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

const monitorType = "trace-forwarder"

var logger = utils.NewThrottledLogger(log.WithFields(log.Fields{"monitorType": monitorType}), 30*time.Second)
var golibLogger = &utils.LogrusGolibShim{FieldLogger: logger.FieldLogger}

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// MONITOR(trace-forwarder): Runs an HTTP server that listens for trace spans
// and forwards them to SignalFx (or the configured ingest host in the `writer`
// section of the agent config).  This supports the same span formats that our
// ingest server supports and at the same path (`/v1/trace`).  By default, the
// server listens on localhost port 9080 but can be configured to anything.

// Config for this monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"false" singleInstance:"true"`
	// The host:port on which to listen for spans.  This server accepts spans
	// in all of the formats that we support on our regular ingest server.  The
	// listening server accepts spans on the same HTTP path that ingest accepts
	// them (e.g. `/v1/trace`).  Requests to other paths will return 404s.
	ListenAddress string `yaml:"listenAddress" default:"127.0.0.1:9080"`
	// HTTP timeout duration for both read and writes. This should be a
	// duration string that is accepted by https://golang.org/pkg/time/#ParseDuration
	ServerTimeout time.Duration `yaml:"serverTimeout" default:"5s"`
	// Whether to send internal metrics about the HTTP listener
	SendInternalMetrics *bool `yaml:"sendInternalMetrics" default:"false"`
}

// Monitor that accepts and forwards trace spans
type Monitor struct {
	Output types.Output
	cancel context.CancelFunc
}

// Configure the monitor and kick off volume metric syncing
func (m *Monitor) Configure(conf *Config) error {
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	listenerMetrics, err := startListeningForSpans(ctx, conf.ListenAddress, conf.ServerTimeout, traceSinkFuncWrapper(m.forwardSpans))
	if err != nil {
		return errors.WithMessage(err, "could not start trace span listener")
	}

	if *conf.SendInternalMetrics {
		utils.RunOnInterval(ctx, func() {
			for _, dp := range listenerMetrics.Datapoints() {
				m.Output.SendDatapoint(dp)
			}
		}, time.Duration(conf.IntervalSeconds)*time.Second)
	}

	return nil
}

func (m *Monitor) forwardSpans(ctx context.Context, spans []*trace.Span) error {
	for i := range spans {
		m.Output.SendSpan(spans[i])
	}
	return nil
}

// Shutdown stops the forwarder and correlation MTSs
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
