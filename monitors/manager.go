package monitors

import (
	"reflect"
	"sync"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/observers"
	log "github.com/sirupsen/logrus"
)

// MonitorManager coordinates the startup and shutdown of monitors based on the
// configuration provided by the user.  Monitors that have discovery rules can
// be injected with multiple services.  If a monitor does not have a discovery
// rule (a "static" monitor), it will be started immediately (as soon as
// Configure is called).
type MonitorManager struct {
	monitorConfigs []*config.MonitorConfig
	// Keep track of which services go with which monitor
	activeMonitors []*ActiveMonitor
	lock           sync.Mutex
	// Map of services that are being actively monitored
	discoveredServices map[observers.ServiceID]*observers.ServiceInstance

	DPChannel    chan<- *datapoint.Datapoint
	EventChannel chan<- *event.Event
}

type ActiveMonitor struct {
	instance   interface{}
	config     *config.MonitorConfig
	serviceSet map[observers.ServiceID]bool
	// Is the monitor marked for deletion?
	doomed bool
}

func (am *ActiveMonitor) Shutdown() {
	if sh, ok := am.instance.(Shutdownable); ok {
		sh.Shutdown()
	}
}

func (mm *MonitorManager) ensureInit() {
	if mm.activeMonitors == nil {
		mm.activeMonitors = make([]*ActiveMonitor, 0)
	}
	if mm.discoveredServices == nil {
		mm.discoveredServices = make(map[observers.ServiceID]*observers.ServiceInstance)
	}
}

func (mm *MonitorManager) Configure(confs []config.MonitorConfig) {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	mm.ensureInit()
	mm.monitorConfigs = make([]*config.MonitorConfig, 0, len(confs))

	// All monitors are marked for deletion at first.  They can be saved and
	// reused by having a compatible config in the newly provided config
	mm.markAllMonitorsAsDoomed()

	for i := range confs {
		conf := &confs[i]

		monConfig := getCustomConfigForMonitor(conf)
		if monConfig == nil {
			continue
		}

		configMatchedActive := false
		for i := range mm.activeMonitors {
			am := mm.activeMonitors[i]
			if am.doomed {
				configEqual := reflect.DeepEqual(*am.config, *conf)
				monitorsCompatible := am.config.Type == conf.Type && am.config.DiscoveryRule == conf.DiscoveryRule
				if monitorsCompatible {
					am.doomed = false
					configMatchedActive = true

					log.WithFields(log.Fields{
						"configEqual":   configEqual,
						"monitorType":   am.config.Type,
						"discoveryRule": am.config.DiscoveryRule,
					}).Debug("Reconfiguration found a compatible monitor that will be reused")

					if !configEqual {
						am.config = conf
						configureMonitor(am.instance, monConfig)
					}
				}
			}
		}

		// No discovery rule means that the monitor should run from the start
		if conf.DiscoveryRule == "" && !configMatchedActive {
			if mm.createAndConfigureNewMonitor(conf) == nil {
				continue
			}
		}

		mm.monitorConfigs = append(mm.monitorConfigs, conf)
	}

	mm.deleteDoomedMonitors()

	for i := range mm.monitorConfigs {
		mm.findMonitorsForDiscoveredServices(mm.monitorConfigs[i])
	}

}

func (mm *MonitorManager) markAllMonitorsAsDoomed() {
	for i := range mm.activeMonitors {
		mm.activeMonitors[i].doomed = true
	}
}

func (mm *MonitorManager) deleteDoomedMonitors() {
	newActiveMonitors := []*ActiveMonitor{}

	for i := range mm.activeMonitors {
		am := mm.activeMonitors[i]
		if am.doomed {
			log.WithFields(log.Fields{
				"serviceSet":    am.serviceSet,
				"monitorType":   am.config.Type,
				"discoveryRule": am.config.DiscoveryRule,
			}).Debug("Shutting down doomed monitor")

			am.Shutdown()
		} else {
			newActiveMonitors = append(newActiveMonitors, am)
		}
	}

	mm.activeMonitors = newActiveMonitors
}

// Does some reflection magic to pass the right type to the Configure method of
// each monitor
func configureMonitor(monitor interface{}, conf interface{}) bool {
	return config.CallConfigure(monitor, conf)
}

func (mm *MonitorManager) findMonitorsForDiscoveredServices(conf *config.MonitorConfig) {
	log.WithFields(log.Fields{
		"discoveredServices": mm.discoveredServices,
	}).Debug("Finding monitors for discovered services")

	for _, service := range mm.discoveredServices {
		log.WithFields(log.Fields{
			"monitorType":   conf.Type,
			"discoveryRule": conf.DiscoveryRule,
			"service":       service,
		}).Debug("Trying to find config that matches discovered service")

		if mm.monitorServiceIfRuleMatches(conf, service) {
			log.WithFields(log.Fields{
				"service":       service,
				"monitorConfig": *conf,
			}).Info("Now monitoring discovered service")
		}
	}
}

// Returns true is the service is now monitored
func (mm *MonitorManager) monitorServiceIfRuleMatches(config *config.MonitorConfig, service *observers.ServiceInstance) bool {
	if config.DiscoveryRule == "" || !doesServiceMatchRule(service, config.DiscoveryRule) {
		return false
	}
	monitor := mm.ensureCompatibleServiceMonitorExists(config)
	if monitor == nil {
		return false
	}

	if _, ok := monitor.serviceSet[service.ID]; ok {
		// Already monitoring this service so don't inject it again to the
		// monitor
		return true
	}

	if !injectServiceToMonitorInstance(monitor, service) {
		monitor.doomed = true
		mm.deleteDoomedMonitors()
		return false
	}

	monitor.serviceSet[service.ID] = true
	return true
}

// ServiceAdded should be called when a new service is discovered
func (mm *MonitorManager) ServiceAdded(service *observers.ServiceInstance) {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	mm.discoveredServices[service.ID] = service

	watching := false
	for _, config := range mm.monitorConfigs {
		watching = mm.monitorServiceIfRuleMatches(config, service) || watching
	}

	if !watching {
		log.WithFields(log.Fields{
			"service": *service,
		}).Debug("Service added that doesn't match any discovery rules")
	}
}

// This ensures a monitor for a particular discovery rule and type exists.  It
// is not meant to be used for static monitors.
func (mm *MonitorManager) ensureCompatibleServiceMonitorExists(config *config.MonitorConfig) *ActiveMonitor {
	// See if we can find an existing compatible monitor
	for i := range mm.activeMonitors {
		am := mm.activeMonitors[i]
		if am.config.Type == config.Type && am.config.DiscoveryRule == config.DiscoveryRule {
			log.WithFields(log.Fields{
				"monitorType":   config.Type,
				"activeMonRule": am.config.DiscoveryRule,
				"inputRule":     config.DiscoveryRule,
			}).Debug("Compatible monitor found, returning")

			return am
		}
	}

	// No compatible monitor found so make a new one
	return mm.createAndConfigureNewMonitor(config)
}

func (mm *MonitorManager) createAndConfigureNewMonitor(config *config.MonitorConfig) *ActiveMonitor {
	instance := newMonitor(config.Type)

	mm.injectDatapointChannelIfNeeded(instance)
	mm.injectEventsChannelIfNeeded(instance)

	monConfig := getCustomConfigForMonitor(config)
	if monConfig == nil {
		return nil
	}

	if !configureMonitor(instance, monConfig) {
		return nil
	}
	am := &ActiveMonitor{
		instance:   instance,
		serviceSet: make(map[observers.ServiceID]bool),
		config:     config,
	}
	mm.activeMonitors = append(mm.activeMonitors, am)

	log.WithFields(log.Fields{
		"monitorType":   config.Type,
		"discoveryRule": config.DiscoveryRule,
	}).Debug("Creating new monitor")

	return am
}

// Sets the `DPs` field on a monitor if it is present to the datapoint channel.
// Returns whether the field was actually set.
func (mm *MonitorManager) injectDatapointChannelIfNeeded(instance interface{}) bool {
	instanceValue := reflect.Indirect(reflect.ValueOf(instance))
	dpsValue := instanceValue.FieldByName("DPs")
	if !dpsValue.IsValid() {
		return false
	}
	if dpsValue.Type() != reflect.ChanOf(reflect.SendDir, reflect.TypeOf(&datapoint.Datapoint{})) {
		log.WithFields(log.Fields{
			"pkgPath": instanceValue.Type().PkgPath(),
		}).Error("Monitor instance has 'DPs' member but is not of type 'chan<- *datapoint.Datapoint'")
		return false
	}
	dpsValue.Set(reflect.ValueOf(mm.DPChannel))

	return true
}

// Sets the `Events` field on a monitor if it is present to the events channel.
// Returns whether the field was actually set.
func (mm *MonitorManager) injectEventsChannelIfNeeded(instance interface{}) bool {
	instanceValue := reflect.Indirect(reflect.ValueOf(instance))
	eventsValue := instanceValue.FieldByName("Events")
	if !eventsValue.IsValid() {
		return false
	}

	if eventsValue.Type() != reflect.ChanOf(reflect.SendDir, reflect.TypeOf(&event.Event{})) {
		log.WithFields(log.Fields{
			"pkgPath": instanceValue.Type().PkgPath(),
		}).Error("Monitor instance has 'Events' member but is not of type 'chan<- *event.Event'")
		return false
	}
	eventsValue.Set(reflect.ValueOf(mm.EventChannel))

	return true
}

func injectServiceToMonitorInstance(monitor *ActiveMonitor, service *observers.ServiceInstance) bool {
	if inst, ok := monitor.instance.(InjectableMonitor); ok {
		inst.AddService(service)
		return true
	}

	log.WithFields(log.Fields{
		"monitor": monitor.instance,
	}).Error("Monitor does not provide the service injection methods!")
	return false
}

func (mm *MonitorManager) monitorsForServiceID(id observers.ServiceID) (out []*ActiveMonitor) {
	for i := range mm.activeMonitors {
		if mm.activeMonitors[i].serviceSet[id] {
			out = append(out, mm.activeMonitors[i])
		}
	}
	return // Named return value
}

func (mm *MonitorManager) ServiceRemoved(service *observers.ServiceInstance) {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	delete(mm.discoveredServices, service.ID)

	monitors := mm.monitorsForServiceID(service.ID)
	for _, am := range monitors {
		removeServiceFromMonitor(am, service)

		delete(am.serviceSet, service.ID)

		if len(am.serviceSet) == 0 {
			log.WithFields(log.Fields{
				"monitorType":   am.config.Type,
				"discoveryRule": am.config.DiscoveryRule,
			}).Info("Monitor no longer has active services, shutting down")

			am.doomed = true
		}
	}
	mm.deleteDoomedMonitors()

	log.WithFields(log.Fields{
		"service": *service,
	}).Info("No longer monitoring service")
}

func removeServiceFromMonitor(monitor *ActiveMonitor, service *observers.ServiceInstance) bool {
	if inst, ok := monitor.instance.(InjectableMonitor); ok {
		inst.RemoveService(service)
		return true
	}

	log.WithFields(log.Fields{
		"service": service,
	}).Error("Monitor does not provide the service injection methods!")
	return false
}

func (mm *MonitorManager) isServiceMonitored(service *observers.ServiceInstance) bool {
	monitors := mm.monitorsForServiceID(service.ID)
	return len(monitors) > 0
}

func (mm *MonitorManager) Shutdown() {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	for i := range mm.activeMonitors {
		mm.activeMonitors[i].doomed = true
	}
	mm.deleteDoomedMonitors()

	mm.activeMonitors = nil
	mm.discoveredServices = nil
}
