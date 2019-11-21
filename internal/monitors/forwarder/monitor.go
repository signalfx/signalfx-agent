package forwarder

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

var logger = utils.NewThrottledLogger(log.WithFields(log.Fields{"monitorType": monitorType}), 30*time.Second)
var golibLogger = &utils.LogrusGolibShim{FieldLogger: logger.FieldLogger}

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"false" singleInstance:"true"`
	// The host:port on which to listen for datapoints.  The listening server
	// accepts datapoints on the same HTTP path that ingest/gateway accepts
	// them (e.g. `/v2/datapoint`, `/v1/trace`).  Requests to other paths will
	// return 404s.
	ListenAddress string `yaml:"listenAddress" default:"127.0.0.1:9080"`
	// HTTP timeout duration for both read and writes. This should be a
	// duration string that is accepted by https://golang.org/pkg/time/#ParseDuration
	ServerTimeout time.Duration `yaml:"serverTimeout" default:"5s"`
	// Whether to emit internal metrics about the HTTP listener
	SendInternalMetrics *bool `yaml:"sendInternalMetrics" default:"false"`
}

// Monitor that accepts and forwards SignalFx data
type Monitor struct {
	Output types.Output
	cancel context.CancelFunc
}

// Configure the monitor and kick off volume metric syncing
func (m *Monitor) Configure(conf *Config) error {
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	sink := &outputSink{Output: m.Output}
	listenerMetrics, err := startListening(ctx, conf.ListenAddress, conf.ServerTimeout, sink)
	if err != nil {
		return errors.WithMessage(err, "could not start forwarder listener")
	}

	if *conf.SendInternalMetrics {
		utils.RunOnInterval(ctx, func() {
			m.Output.SendDatapoints(listenerMetrics.Datapoints()...)
		}, time.Duration(conf.IntervalSeconds)*time.Second)
	}

	return nil
}

// Shutdown stops the forwarder and correlation MTSs
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
