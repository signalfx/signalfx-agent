// Package monitors is the core logic for monitors.  Monitors are what collect
// metrics from the environment.  They have a simple interface that all must
// implement: the Configure method, which takes one argument of the same type
// that you pass as the configTemplate to the Register function.  Optionally,
// monitors may implement the niladic Shutdown method to do cleanup.  Monitors
// will never be reused after the Shutdown method is called.
//
// If your monitor is used for dynamically discovered services, you should
// implement the InjectableMonitor interface, which simply includes two
// methods that are called when services are added and removed.
//
// If a monitor wants to create SignalFx golib datapoints/events and have them
// sent by the agent.  The monitor type should define a "DPs" and/or "Events"
// field of the type "chan<- datapoints.Datapoint" and "chan<- events.Event".
// The monitor manager will automatically inject those fields before Configure
// is called.  They could be swapped out at any time, so monitors should not
// cache those fields in other variables.
package monitors

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/utils"
	log "github.com/sirupsen/logrus"
)

// MonitorFactory is a niladic function that creates an unconfigured instance
// of a monitor.
type MonitorFactory func() interface{}

// MonitorFactories holds all of the registered monitor factories
var MonitorFactories = map[string]MonitorFactory{}

// These are blank (zero-value) instances of the configuration struct for a
// particular monitor type.
var configTemplates = map[string]config.MonitorCustomConfig{}

// InjectableMonitor should be implemented by a dynamic monitor that needs to
// know when services are added and removed.
type InjectableMonitor interface {
	AddService(services.Endpoint)
	RemoveService(services.Endpoint)
}

// Register a new monitor type with the agent.  This is intended to be called
// from the init function of the module of a specific monitor
// implementation. configTemplate should be a zero-valued struct that is of the
// same type as the parameter to the Configure method for this monitor type.
func Register(_type string, factory MonitorFactory, configTemplate config.MonitorCustomConfig) {
	if _, ok := MonitorFactories[_type]; ok {
		panic("Monitor type '" + _type + "' already registered")
	}
	MonitorFactories[_type] = factory
	configTemplates[_type] = configTemplate
}

// DeregisterAll unregisters all monitor types.  Primarily intended for testing
// purposes.
func DeregisterAll() {
	for k := range MonitorFactories {
		delete(MonitorFactories, k)
	}

	for k := range configTemplates {
		delete(configTemplates, k)
	}
}

func newUninitializedMonitor(_type string) interface{} {
	if factory, ok := MonitorFactories[_type]; ok {
		return factory()
	}

	log.WithFields(log.Fields{
		"monitorType": _type,
	}).Error("Monitor type not supported")
	return nil
}

// Creates a new, unconfigured instance of a monitor of _type.  Returns nil if
// the monitor type is not registered.
func newMonitor(_type string) interface{} {
	mon := newUninitializedMonitor(_type)
	if initMon, ok := mon.(Initializable); ok {
		if err := initMon.Init(); err != nil {
			log.WithFields(log.Fields{
				"error":       err,
				"monitorType": _type,
			}).Error("Could not initialize monitor")
			return nil
		}
	}
	return mon

}

// Initializable represents a monitor that has a distinct InitMonitor method.
// This should be called once after the monitor is created and before any of
// its other methods are called.  It is useful for things that are not
// appropriate to do in the monitor factory function.
type Initializable interface {
	Init() error
}

// Shutdownable should be implemented by all monitors that need to clean up
// resources before being destroyed.
type Shutdownable interface {
	Shutdown()
}

// Takes a generic MonitorConfig and pulls out monitor-specific config to
// populate a clone of the config template that was registered for the monitor
// type specified in conf.  This will also validate the config and return nil
// if validation fails.
func getCustomConfigForMonitor(conf *config.MonitorConfig) (config.MonitorCustomConfig, bool) {
	confTemplate, ok := configTemplates[conf.Type]
	if !ok {
		log.WithFields(log.Fields{
			"monitorType": conf.Type,
		}).Error("Unknown monitor type")
		return nil, false
	}
	monConfig := utils.CloneInterface(confTemplate).(config.MonitorCustomConfig)

	if ok := config.FillInConfigTemplate("MonitorConfig", monConfig, conf); !ok {
		return nil, false
	}

	// These methods will set state inside the config such that conf.IsValid
	// will return true or false
	if err := validateConfig(monConfig); err != nil {
		monConfig.CoreConfig().ValidationError = err.Error()
		return monConfig, false
	}

	return monConfig, true
}

func anyMarkedSolo(confs []config.MonitorConfig) bool {
	for i := range confs {
		if confs[i].Solo {
			return true
		}
	}
	return false
}
