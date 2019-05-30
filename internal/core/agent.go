// Package core contains the central frame of the agent that hooks up the
// various subsystems.
package core

import (
	"context"
	"io"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/golib/trace"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/core/hostid"
	"github.com/signalfx/signalfx-agent/internal/core/meta"
	"github.com/signalfx/signalfx-agent/internal/core/services"
	"github.com/signalfx/signalfx-agent/internal/core/writer"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/observers"
)

const (
	// Items should stay in these channels only very briefly as there should be
	// goroutines dedicated to pulling them at all times.  Having the capacity
	// be non-zero is more an optimization to keep monitors from seizing up
	// under extremely heavy load.
	datapointChanCapacity = 3000
	eventChanCapacity     = 100
	dimPropChanCapacity   = 100
	traceSpanChanCapacity = 3000
)

// Agent is what hooks up observers, monitors, and the datapoint writer.
type Agent struct {
	observers    *observers.ObserverManager
	monitors     *monitors.MonitorManager
	writer       *writer.SignalFxWriter
	meta         *meta.AgentMeta
	lastConfig   *config.Config
	dpChan       chan *datapoint.Datapoint
	eventChan    chan *event.Event
	propertyChan chan *types.DimProperties
	spanChan     chan *trace.Span

	diagnosticServer     *http.Server
	profileServerRunning bool
	startTime            time.Time
}

// NewAgent creates an unconfigured agent instance
func NewAgent() *Agent {
	agent := Agent{
		dpChan:       make(chan *datapoint.Datapoint, datapointChanCapacity),
		eventChan:    make(chan *event.Event, eventChanCapacity),
		propertyChan: make(chan *types.DimProperties, dimPropChanCapacity),
		spanChan:     make(chan *trace.Span, traceSpanChanCapacity),
		startTime:    time.Now(),
	}

	agent.observers = &observers.ObserverManager{
		CallbackTargets: &observers.ServiceCallbacks{
			Added:   agent.endpointAdded,
			Removed: agent.endpointRemoved,
		},
	}

	agent.meta = &meta.AgentMeta{}
	agent.monitors = monitors.NewMonitorManager(agent.meta)
	agent.monitors.DPs = agent.dpChan
	agent.monitors.Events = agent.eventChan
	agent.monitors.DimensionProps = agent.propertyChan
	agent.monitors.TraceSpans = agent.spanChan
	return &agent
}

func (a *Agent) configure(conf *config.Config) {
	log.SetFormatter(conf.Logging.LogrusFormatter())

	level := conf.Logging.LogrusLevel()
	if level != nil {
		log.SetLevel(*level)
	}

	log.Infof("Using log level %s", log.GetLevel().String())

	if !conf.DisableHostDimensions {
		conf.Writer.HostIDDims = hostid.Dimensions(conf.SendMachineID, conf.Hostname, conf.UseFullyQualifiedHost)
	}

	if conf.EnableProfiling {
		a.ensureProfileServerRunning(conf.ProfilingHost, conf.ProfilingPort)
	}

	if a.lastConfig == nil || a.lastConfig.Writer.Hash() != conf.Writer.Hash() {
		if a.writer != nil {
			a.writer.Shutdown()
		}
		var err error
		a.writer, err = writer.New(&conf.Writer, a.dpChan, a.eventChan, a.propertyChan, a.spanChan)
		if err != nil {
			// This is a catastrophic error if we can't write datapoints.
			log.WithError(err).Error("Could not configure SignalFx datapoint writer, unable to start up")
			os.Exit(4)
		}
	}

	a.meta.InternalStatusHost = conf.InternalStatusHost
	a.meta.InternalStatusPort = conf.InternalStatusPort

	// The order of Configure calls is very important!
	a.monitors.Configure(conf.Monitors, &conf.Collectd, conf.IntervalSeconds, conf.EnableBuiltInFiltering)
	a.observers.Configure(conf.Observers)
	a.lastConfig = conf
}

func (a *Agent) endpointAdded(service services.Endpoint) {
	a.monitors.EndpointAdded(service)
}

func (a *Agent) endpointRemoved(service services.Endpoint) {
	a.monitors.EndpointRemoved(service)
}

func (a *Agent) shutdown() {
	a.observers.Shutdown()
	a.monitors.Shutdown()
	//neopy.Instance().Shutdown()
	a.writer.Shutdown()
}

// Startup the agent.  Returns a function that can be called to shutdown the
// agent, as well as a channel that will be notified when the agent has
// shutdown.
func Startup(configPath string) (context.CancelFunc, <-chan struct{}) {
	cwc, cancel := context.WithCancel(context.Background())

	configLoads, err := config.LoadConfig(cwc, configPath)
	if err != nil {
		log.WithFields(log.Fields{
			"error":      err,
			"configPath": configPath,
		}).Error("Error loading main config")
		os.Exit(1)
	}

	agent := NewAgent()

	shutdownComplete := make(chan struct{})

	go func(ctx context.Context) {
		for {
			select {
			case config := <-configLoads:
				log.Info("New config loaded")

				if config == nil {
					log.WithFields(log.Fields{
						"path": configPath,
					}).Error("Failed to load config, cannot continue!")
					os.Exit(2)
				}

				agent.configure(config)
				log.Info("Done configuring agent")

				if config.InternalStatusHost != "" {
					agent.serveDiagnosticInfo(config.InternalStatusHost, config.InternalStatusPort)
				}

			case <-ctx.Done():
				agent.shutdown()
				close(shutdownComplete)
				return
			}
		}
	}(cwc)

	return cancel, shutdownComplete
}

// Status reads the text from the diagnostic socket and returns it if available.
func Status(configPath string, section string) ([]byte, error) {
	configLoads, err := config.LoadConfig(context.Background(), configPath)
	if err != nil {
		return nil, err
	}

	conf := <-configLoads
	return readStatusInfo(conf.InternalStatusHost, conf.InternalStatusPort, section)
}

// StreamDatapoints reads the text from the diagnostic socket and returns it if available.
func StreamDatapoints(configPath string, metric string, dims string) (io.ReadCloser, error) {
	configLoads, err := config.LoadConfig(context.Background(), configPath)
	if err != nil {
		return nil, err
	}

	conf := <-configLoads
	return streamDatapoints(conf.InternalStatusHost, conf.InternalStatusPort, metric, dims)
}
