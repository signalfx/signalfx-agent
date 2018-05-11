// Package core contains the central frame of the agent that hooks up the
// various subsystems.
package core

import (
	"context"
	"errors"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/core/hostid"
	"github.com/signalfx/signalfx-agent/internal/core/meta"
	"github.com/signalfx/signalfx-agent/internal/core/services"
	"github.com/signalfx/signalfx-agent/internal/core/writer"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/observers"
)

// Agent is what hooks up observers, monitors, and the datapoint writer.
type Agent struct {
	observers  *observers.ObserverManager
	monitors   *monitors.MonitorManager
	writer     *writer.SignalFxWriter
	meta       *meta.AgentMeta
	lastConfig *config.Config

	diagnosticServerStop      func()
	internalMetricsServerStop func()
	profileServerRunning      bool
}

// NewAgent creates an unconfigured agent instance
func NewAgent() *Agent {
	agent := Agent{}

	agent.observers = &observers.ObserverManager{
		CallbackTargets: &observers.ServiceCallbacks{
			Added:   agent.endpointAdded,
			Removed: agent.endpointRemoved,
		},
	}

	agent.writer = writer.New()
	agent.meta = &meta.AgentMeta{}
	agent.monitors = monitors.NewMonitorManager(agent.meta)
	return &agent
}

func (a *Agent) configure(conf *config.Config) {
	level := conf.Logging.LogrusLevel()
	if level != nil {
		log.SetLevel(*level)
	}
	log.Infof("Using log level %s", log.GetLevel().String())

	if !conf.DisableHostDimensions {
		a.writer.HostIDDims = hostid.Dimensions(conf.SendMachineID, conf.Hostname, conf.UseFullyQualifiedHost)
	}

	if conf.EnableProfiling {
		a.ensureProfileServerRunning()
	}

	if err := a.writer.Configure(&conf.Writer); err != nil {
		// This is a catastrophic error if we can't write datapoints.
		log.WithError(err).Error("Could not configure SignalFx datapoint writer, unable to start up")
		os.Exit(4)
	}

	// These channels should only be initialized once and never change for the
	// lifetime of the agent.  This means that buffer size changes in the
	// config require a restart.
	if a.monitors.DPs == nil {
		a.monitors.DPs = a.writer.DPChannel()
	}
	if a.monitors.Events == nil {
		a.monitors.Events = a.writer.EventChannel()
	}
	if a.monitors.DimensionProps == nil {
		a.monitors.DimensionProps = a.writer.DimPropertiesChannel()
	}

	a.meta.InternalMetricsServerPath = conf.InternalMetricsServerPath

	//if conf.PythonEnabled {
	//neopy.Instance().Configure()
	//neopy.Instance().EnsureMonitorsRegistered()
	//} else if a.lastConfig != nil && a.lastConfig.PythonEnabled {
	//neopy.Instance().Shutdown()
	//}

	// The order of Configure calls is very important!
	a.monitors.Configure(conf.Monitors, &conf.Collectd, conf.IntervalSeconds)
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

				if config.DiagnosticsServerPath != "" {
					if err := agent.serveDiagnosticInfo(config.DiagnosticsServerPath); err != nil {
						log.WithError(err).Error("Could not start diagnostic socket")
					}
				}
				if config.InternalMetricsServerPath != "" {
					if err := agent.serveInternalMetrics(config.InternalMetricsServerPath); err != nil {
						log.WithError(err).Error("Could not start internal metrics socket")
					}
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
func Status(configPath string) ([]byte, error) {
	configLoads, err := config.LoadConfig(context.Background(), configPath)
	if err != nil {
		return nil, err
	}

	select {
	case conf := <-configLoads:
		if conf.DiagnosticsServerPath == "" {
			return nil, errors.New("diagnosticsSocketPath is blank so cannot get status")
		}
		return readDiagnosticInfo(conf.DiagnosticsServerPath)
	}
}
