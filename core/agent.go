package core

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/config/stores"
	"github.com/signalfx/neo-agent/core/writer"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
	"github.com/signalfx/neo-agent/monitors/neopy"
	"github.com/signalfx/neo-agent/observers"
)

// Agent is what hooks up observers and monitors
type Agent struct {
	observers        *observers.ObserverManager
	monitors         *monitors.MonitorManager
	lastCollectdConf config.CollectdConfig
}

// New creates a agent instance
func NewAgent() *Agent {
	agent := Agent{}

	agent.observers = &observers.ObserverManager{
		CallbackTargets: &observers.ServiceCallbacks{
			Added:   agent.ServiceAdded,
			Removed: agent.ServiceRemoved,
		},
	}

	agent.monitors = &monitors.MonitorManager{}

	return &agent
}

func (a *Agent) Configure(conf *config.Config) {
	level := conf.Logging.LogrusLevel()
	if level != nil {
		log.SetLevel(*level)
	}
	log.Infof("Using log level %s", log.GetLevel().String())

	writer, err := writer.New(conf.IngestURLAsURL(), conf.SignalFxAccessToken)
	if err != nil {
		// There isn't really much we can do at this point so return and wait
		// for another configuration to come in.
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Could not create SignalFx datapoint writer, unable to start up")
		os.Exit(4)
	}
	writer.ListenForDatapoints()

	neopy.GetInstance().Configure()
	neopy.GetInstance().EnsureMonitorsRegistered()

	// The order of Configure calls is very important!
	collectd.CollectdSingleton.Configure(&conf.Collectd)
	a.monitors.Configure(conf.Monitors, writer.DPChannel(), writer.EventChannel())
	a.observers.Configure(conf.Observers)
}

func (a *Agent) ServiceAdded(service *observers.ServiceInstance) {
	monitors.EnsureProxyingDisabledForService(service)
	a.monitors.ServiceAdded(service)
}

func (a *Agent) ServiceRemoved(service *observers.ServiceInstance) {
	a.monitors.ServiceRemoved(service)
}

func (a *Agent) Shutdown() {
	a.observers.Shutdown()
	a.monitors.Shutdown()
	collectd.CollectdSingleton.Shutdown()
	neopy.GetInstance().Shutdown()
}

// Startup the agent.  Returns a function that can be called to shutdown the
// agent, as well as a channel that will be notified when the agent has
// shutdown.
func Startup(configPath string) (context.CancelFunc, <-chan struct{}) {
	log.Debug("Starting up agent")

	cwc, cancel := context.WithCancel(context.Background())

	metaStore := stores.NewMetaStore()

	// Configure the config store from envvars so that we can load config from
	// a non-fs based config store.
	metaStore.ConfigureFromEnv()

	configLoads, err := metaStore.WatchPath(configPath)
	if err != nil {
		log.WithFields(log.Fields{
			"error":      err,
			"configPath": configPath,
		}).Error("Error loading main config")
		os.Exit(1)
	}

	agent := NewAgent()

	exitedCh := make(chan struct{})

	go func(ctx context.Context) {
		for {
			select {
			case configKVPair := <-configLoads:
				log.Debug("Config loaded")

				if configKVPair.Value == nil {
					log.WithFields(log.Fields{
						"path": configPath,
					}).Error("Could not load config file!")
					os.Exit(1)
				} else if len(configKVPair.Value) == 0 {
					log.WithFields(log.Fields{
						"path": configPath,
					}).Error("Config file is blank!")
					os.Exit(1)
				}

				conf, err := config.LoadConfigFromContent(configKVPair.Value)
				if err != nil || conf == nil {
					log.WithFields(log.Fields{
						"path": configPath,
					}).Error("Failed to load config, cannot continue!")
					os.Exit(2)
				}

				log.Debug("Configuring agent")
				agent.Configure(conf)

			case <-ctx.Done():
				agent.Shutdown()
				metaStore.Close()
				exitedCh <- struct{}{}
				return
			}
		}
	}(cwc)

	return cancel, exitedCh
}
