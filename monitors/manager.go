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
	activeMonitorsByServiceID map[observers.ServiceID]*ActiveMonitor
	activeMonitorsByType      map[string][]*ActiveMonitor
	lock                      sync.Mutex
	// Map of services that are being actively monitored
	monitoredServices map[observers.ServiceID]*observers.ServiceInstance
	// A list of services that have been picked up by observers but did not
	// match any configured monitor, or were abandoned by monitors that are
	// shutdown.  These are remembered to solve an edge case in which the agent
	// is hot-reconfigured with monitors whose discovery rule will now match
	// one of these services.
	orphanedServices map[observers.ServiceID]*observers.ServiceInstance
	dpChannel        chan<- *datapoint.Datapoint
	eventChannel     chan<- *event.Event
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
	if mm.activeMonitorsByServiceID == nil {
		mm.activeMonitorsByServiceID = make(map[observers.ServiceID]*ActiveMonitor)
	}
	if mm.activeMonitorsByType == nil {
		mm.activeMonitorsByType = make(map[string][]*ActiveMonitor)
	}
	if mm.monitoredServices == nil {
		mm.monitoredServices = make(map[observers.ServiceID]*observers.ServiceInstance)
	}
	if mm.orphanedServices == nil {
		mm.orphanedServices = make(map[observers.ServiceID]*observers.ServiceInstance)
	}
}

func (mm *MonitorManager) Configure(
	confs []config.MonitorConfig,
	dpChannel chan<- *datapoint.Datapoint,
	eventChannel chan<- *event.Event) {

	mm.lock.Lock()
	defer mm.lock.Unlock()

	mm.dpChannel = dpChannel
	mm.eventChannel = eventChannel

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
		mm.iterateActive(func(am *ActiveMonitor) {
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
		})

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
		mm.findMonitorsForOrphans(mm.monitorConfigs[i])
	}

}

// Accepts a function that will be called for each monitor currently active
func (mm *MonitorManager) iterateActive(it func(*ActiveMonitor)) {
	for i := range mm.activeMonitorsByType {
		for j := range mm.activeMonitorsByType[i] {
			it(mm.activeMonitorsByType[i][j])
		}
	}
}

func (mm *MonitorManager) markAllMonitorsAsDoomed() {
	mm.iterateActive(func(am *ActiveMonitor) {
		am.doomed = true
	})
}

func (mm *MonitorManager) deleteDoomedMonitors() {
	// Delete it from both active monitor service and type map
	for id := range mm.activeMonitorsByServiceID {
		am := mm.activeMonitorsByServiceID[id]
		if am.doomed {
			delete(mm.activeMonitorsByServiceID, id)
		}
	}

	newActiveMonitors := map[string][]*ActiveMonitor{}
	mm.iterateActive(func(am *ActiveMonitor) {
		if am.doomed {
			log.WithFields(log.Fields{
				"serviceSet":    am.serviceSet,
				"monitorType":   am.config.Type,
				"discoveryRule": am.config.DiscoveryRule,
			}).Debug("Shutting down doomed monitor")

			for sid := range am.serviceSet {
				if service, ok := mm.monitoredServices[sid]; ok {
					delete(mm.monitoredServices, sid)
					mm.orphanedServices[sid] = service
				}
			}
			am.Shutdown()
		} else {
			newActiveMonitors[am.config.Type] = append(newActiveMonitors[am.config.Type], am)
		}
	})
	mm.activeMonitorsByType = newActiveMonitors
}

// Does some reflection magic to pass the right type to the Configure method of
// each monitor
func configureMonitor(monitor interface{}, conf interface{}) bool {
	return config.CallConfigure(monitor, conf)
}

func (mm *MonitorManager) findMonitorsForOrphans(conf *config.MonitorConfig) {
	for sid, service := range mm.orphanedServices {
		log.WithFields(log.Fields{
			"monitorType":   conf.Type,
			"discoveryRule": conf.DiscoveryRule,
			"service":       service,
		}).Debug("Trying to find config that matches orphaned service")

		if mm.monitorServiceIfRuleMatches(conf, service) {
			log.WithFields(log.Fields{
				"service":       service,
				"monitorConfig": *conf,
			}).Info("Now monitoring orphaned service")

			delete(mm.orphanedServices, sid)
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

	if !injectServiceToMonitorInstance(monitor, service) {
		monitor.doomed = true
		mm.deleteDoomedMonitors()
		return false
	}

	mm.activeMonitorsByServiceID[service.ID] = monitor
	monitor.serviceSet[service.ID] = true
	mm.monitoredServices[service.ID] = service
	return true
}

// ServiceAdded should be called when a new service is discovered
func (mm *MonitorManager) ServiceAdded(service *observers.ServiceInstance) {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	for _, config := range mm.monitorConfigs {
		if mm.monitorServiceIfRuleMatches(config, service) {
			return
		}
	}
	log.WithFields(log.Fields{
		"service": *service,
	}).Debug("Service added that doesn't match any discovery rules")

	mm.orphanedServices[service.ID] = service
}

// This ensures a monitor for a particular discovery rule and type exists.  It
// is not meant to be used for static monitors.
func (mm *MonitorManager) ensureCompatibleServiceMonitorExists(config *config.MonitorConfig) *ActiveMonitor {
	// See if we can find an existing compatible monitor
	for _, mon := range mm.activeMonitorsByType[config.Type] {
		if mon.config.DiscoveryRule == config.DiscoveryRule {
			log.WithFields(log.Fields{
				"monitorType":   config.Type,
				"activeMonRule": mon.config.DiscoveryRule,
				"inputRule":     config.DiscoveryRule,
			}).Debug("Compatible monitor found, returning")

			return mon
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
	mm.activeMonitorsByType[config.Type] = append(mm.activeMonitorsByType[config.Type], am)

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
	dpsValue.Set(reflect.ValueOf(mm.dpChannel))

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
	eventsValue.Set(reflect.ValueOf(mm.eventChannel))

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

func (mm *MonitorManager) ServiceRemoved(service *observers.ServiceInstance) {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	log.WithFields(log.Fields{
		"service": *service,
	}).Info("No longer monitoring service")

	delete(mm.orphanedServices, service.ID)
	delete(mm.monitoredServices, service.ID)

	if am := mm.activeMonitorsByServiceID[service.ID]; am != nil {
		removeServiceFromMonitor(am, service)

		delete(mm.activeMonitorsByServiceID, service.ID)
		delete(am.serviceSet, service.ID)

		if len(am.serviceSet) == 0 {
			log.WithFields(log.Fields{
				"monitorType":   am.config.Type,
				"discoveryRule": am.config.DiscoveryRule,
			}).Info("Monitor no longer has active services, shutting down")

			am.doomed = true
			mm.deleteDoomedMonitors()
		}
	}
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

func (mm *MonitorManager) Shutdown() {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	mm.iterateActive(func(am *ActiveMonitor) {
		am.doomed = true
	})
	mm.deleteDoomedMonitors()

	mm.activeMonitorsByServiceID = nil
	mm.activeMonitorsByType = nil
	mm.orphanedServices = nil
	mm.monitoredServices = nil
}
