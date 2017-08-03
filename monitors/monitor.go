// Monitors are what collect metrics for specific services.  Monitors have a
// simple interface that all must implement: the Configure method, which takes
// one argument of the same type that you pass as the `configTemplate` to the
// `Register` function.
// If your monitor is used for dynamically discovered services, you should
// implement the `InjectableMonitor` interface, which simply includes two
// methods that are called when services are added and removed.
package monitors

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/observers"
	"github.com/signalfx/neo-agent/utils"
	log "github.com/sirupsen/logrus"
)

type MonitorID string

// MonitorFactory creates an unconfigured instance of an monitor
type MonitorFactory func() interface{}

var MonitorFactories = map[string]MonitorFactory{}

// These are blank (zero-value) instances of the configuration struct for a
// particular monitor type.
var configTemplates = map[string]interface{}{}

// InjectableMonitor is a dynamic monitor that needs to know when services are
// added and removed.
type InjectableMonitor interface {
	AddService(*observers.ServiceInstance)
	RemoveService(*observers.ServiceInstance)
}

// Register a new monitor.  This is intended to be called from the `init`
// function of the package of a specific monitor implementation.
// `configValidator` is a function that receives a config.MonitorConfig and
// returns nil if the config is invalid.
func Register(_type string, factory MonitorFactory, configTemplate interface{}) {
	if _, ok := MonitorFactories[_type]; ok {
		panic("Monitor type '" + _type + "' already registered")
	}
	MonitorFactories[_type] = factory
	configTemplates[_type] = configTemplate
}

func DeregisterAll() {
	for k := range MonitorFactories {
		delete(MonitorFactories, k)
	}

	for k := range configTemplates {
		delete(configTemplates, k)
	}
}

func newMonitor(_type string) interface{} {
	if factory, ok := MonitorFactories[_type]; ok {
		return factory()
	} else {
		log.WithFields(log.Fields{
			"monitorType": _type,
		}).Error("Monitor type not supported")
	}
	return nil
}

type Shutdownable interface {
	Shutdown()
}

func getCustomConfigForMonitor(conf *config.MonitorConfig) interface{} {
	confTemplate, ok := configTemplates[conf.Type]
	if !ok {
		log.WithFields(log.Fields{
			"monitorType": conf.Type,
		}).Error("Unknown monitor type")
		return nil
	}
	monConfig := utils.CloneInterface(confTemplate)

	if ok := config.FillInConfigTemplate("MonitorConfig", monConfig, conf); !ok {
		return nil
	}

	if !validateCommonConfig(conf) || !validateCustomConfig(&monConfig) {
		log.WithFields(log.Fields{
			"monitorType": conf.Type,
		}).Error("Monitor config is invalid, not enabling")
		return nil
	}
	return monConfig
}
