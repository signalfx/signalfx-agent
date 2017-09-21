package monitors

import (
	"reflect"
	"sync"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/core/writer"
	log "github.com/sirupsen/logrus"
)

// MonitorManager coordinates the startup and shutdown of monitors based on the
// configuration provided by the user.  Monitors that have discovery rules can
// be injected with multiple services.  If a monitor does not have a discovery
// rule (a "static" monitor), it will be started immediately (as soon as
// Configure is called).
type MonitorManager struct {
	monitorConfigs []config.MonitorCustomConfig
	// Keep track of which services go with which monitor
	activeMonitors []*ActiveMonitor
	lock           sync.Mutex
	// Map of services that are being actively monitored
	discoveredServices map[services.ID]services.Endpoint

	dpChan      chan<- *datapoint.Datapoint
	eventChan   chan<- *event.Event
	dimPropChan chan<- *writer.DimProperties
}

func (mm *MonitorManager) ensureInit() {
	if mm.activeMonitors == nil {
		mm.activeMonitors = make([]*ActiveMonitor, 0)
	}
	if mm.discoveredServices == nil {
		mm.discoveredServices = make(map[services.ID]services.Endpoint)
	}
}

// Configure receives a list of monitor configurations.  It will start up any
// static monitors and watch discovered services to see if any match dynamic
// monitors.
func (mm *MonitorManager) Configure(confs []config.MonitorConfig) {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	mm.ensureInit()
	mm.monitorConfigs = make([]config.MonitorCustomConfig, 0, len(confs))

	requireSoloTrue := anyMarkedSolo(confs)

	// All monitors are marked for deletion at first.  They can be saved and
	// reused by having a compatible config in the newly provided config
	mm.markAllMonitorsAsDoomed()

	for i := range confs {
		conf := &confs[i]

		monConfig := getCustomConfigForMonitor(conf)
		if monConfig == nil {
			continue
		}

		if requireSoloTrue && !conf.Solo {
			log.Infof("Solo mode is active, skipping monitor of type %s", conf.Type)
			continue
		}

		configMatchedActive := false
		for i := range mm.activeMonitors {
			am := mm.activeMonitors[i]
			if am.doomed {
				coreConfig := am.config.CoreConfig()
				configEqual := reflect.DeepEqual(coreConfig, *conf)
				monitorsCompatible := coreConfig.Type == conf.Type && coreConfig.DiscoveryRule == conf.DiscoveryRule
				if monitorsCompatible {
					configMatchedActive = true
					log.WithFields(log.Fields{
						"configEqual":   configEqual,
						"monitorType":   coreConfig.Type,
						"discoveryRule": coreConfig.DiscoveryRule,
					}).Debug("Reconfiguration found a compatible monitor that will be reused")

					if !configEqual {
						if !am.configureMonitor(monConfig) {
							continue
						}
					}
					am.doomed = false
				}
			}
		}

		// No discovery rule means that the monitor should run from the start
		if conf.DiscoveryRule == "" && !configMatchedActive {
			if mm.createAndConfigureNewMonitor(monConfig) == nil {
				continue
			}
		}

		mm.monitorConfigs = append(mm.monitorConfigs, monConfig)
	}

	mm.deleteDoomedMonitors()

	for i := range mm.monitorConfigs {
		mm.findMonitorsForDiscoveredServices(mm.monitorConfigs[i])
	}

}

// SetDPChannel allows you to inject a new datapoint channel to the manager and
// have it propagated to all active monitors
func (mm *MonitorManager) SetDPChannel(dpChan chan<- *datapoint.Datapoint) {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	mm.dpChan = dpChan

	for i := range mm.activeMonitors {
		mm.activeMonitors[i].injectDatapointChannelIfNeeded(dpChan)
	}
}

// SetEventChannel allows you to inject a new event channel to the manager and
// have it propagated to all active monitors
func (mm *MonitorManager) SetEventChannel(eventChan chan<- *event.Event) {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	mm.eventChan = eventChan

	for i := range mm.activeMonitors {
		mm.activeMonitors[i].injectEventChannelIfNeeded(eventChan)
	}
}

// SetDimPropChannel allows you to inject a new dimension property channel to
// the manager and have it propagated to all active monitors
func (mm *MonitorManager) SetDimPropChannel(dimPropChan chan<- *writer.DimProperties) {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	mm.dimPropChan = dimPropChan

	for i := range mm.activeMonitors {
		mm.activeMonitors[i].injectDimPropertiesChannelIfNeeded(dimPropChan)
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
				"monitorType":   am.config.CoreConfig().Type,
				"discoveryRule": am.config.CoreConfig().DiscoveryRule,
			}).Debug("Shutting down doomed monitor")

			am.Shutdown()
		} else {
			newActiveMonitors = append(newActiveMonitors, am)
		}
	}

	mm.activeMonitors = newActiveMonitors
}

func (mm *MonitorManager) findMonitorsForDiscoveredServices(conf config.MonitorCustomConfig) {
	log.WithFields(log.Fields{
		"discoveredServices": mm.discoveredServices,
	}).Debug("Finding monitors for discovered services")

	for _, service := range mm.discoveredServices {
		log.WithFields(log.Fields{
			"monitorType":   conf.CoreConfig().Type,
			"discoveryRule": conf.CoreConfig().DiscoveryRule,
			"service":       service,
		}).Debug("Trying to find config that matches discovered service")

		if mm.monitorServiceIfRuleMatches(conf, service) {
			log.WithFields(log.Fields{
				"serviceID":   service.ID(),
				"serviceHost": service.Hostname(),
				"monitorType": conf.CoreConfig().Type,
			}).Info("Now monitoring discovered service")
		}
	}
}

// Returns true is the service is now monitored
func (mm *MonitorManager) monitorServiceIfRuleMatches(config config.MonitorCustomConfig, service services.Endpoint) bool {
	if config.CoreConfig().DiscoveryRule == "" || !services.DoesServiceMatchRule(service, config.CoreConfig().DiscoveryRule) {
		return false
	}
	monitor := mm.ensureCompatibleServiceMonitorExists(config)
	if monitor == nil {
		return false
	}

	if _, ok := monitor.serviceSet[service.ID()]; ok {
		// Already monitoring this service so don't inject it again to the
		// monitor
		return true
	}

	if !monitor.injectServiceToMonitorInstance(service) {
		monitor.doomed = true
		mm.deleteDoomedMonitors()
		return false
	}
	return true
}

// ServiceAdded should be called when a new service is discovered
func (mm *MonitorManager) ServiceAdded(service services.Endpoint) {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	ensureProxyingDisabledForService(service)
	mm.discoveredServices[service.ID()] = service

	watching := false
	for _, config := range mm.monitorConfigs {
		watching = mm.monitorServiceIfRuleMatches(config, service) || watching
	}

	if !watching {
		log.WithFields(log.Fields{
			"service": service,
		}).Debug("Service added that doesn't match any discovery rules")
	}
}

// This ensures a monitor for a particular discovery rule and type exists.  It
// is not meant to be used for static monitors.
func (mm *MonitorManager) ensureCompatibleServiceMonitorExists(config config.MonitorCustomConfig) *ActiveMonitor {
	// See if we can find an existing compatible monitor
	for i := range mm.activeMonitors {
		am := mm.activeMonitors[i]
		if am.config.CoreConfig().Type == config.CoreConfig().Type && am.config.CoreConfig().DiscoveryRule == config.CoreConfig().DiscoveryRule {
			log.WithFields(log.Fields{
				"monitorType":   config.CoreConfig().Type,
				"activeMonRule": am.config.CoreConfig().DiscoveryRule,
				"inputRule":     config.CoreConfig().DiscoveryRule,
			}).Debug("Compatible monitor found, returning")

			return am
		}
	}

	// No compatible monitor found so make a new one
	return mm.createAndConfigureNewMonitor(config)
}

func (mm *MonitorManager) createAndConfigureNewMonitor(config config.MonitorCustomConfig) *ActiveMonitor {
	instance := newMonitor(config.CoreConfig().Type)
	if instance == nil {
		return nil
	}

	am := &ActiveMonitor{
		instance:   instance,
		serviceSet: make(map[services.ID]services.Endpoint),
	}

	am.injectDatapointChannelIfNeeded(mm.dpChan)
	am.injectEventChannelIfNeeded(mm.eventChan)
	am.injectDimPropertiesChannelIfNeeded(mm.dimPropChan)

	if !am.configureMonitor(config) {
		return nil
	}
	mm.activeMonitors = append(mm.activeMonitors, am)

	log.WithFields(log.Fields{
		"monitorType":   config.CoreConfig().Type,
		"discoveryRule": config.CoreConfig().DiscoveryRule,
	}).Debug("Creating new monitor")

	return am
}

func (mm *MonitorManager) monitorsForServiceID(id services.ID) (out []*ActiveMonitor) {
	for i := range mm.activeMonitors {
		if mm.activeMonitors[i].serviceSet[id] != nil {
			out = append(out, mm.activeMonitors[i])
		}
	}
	return // Named return value
}

// ServiceRemoved should be called by observers when a service endpoint was
// removed.
func (mm *MonitorManager) ServiceRemoved(service services.Endpoint) {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	delete(mm.discoveredServices, service.ID())

	monitors := mm.monitorsForServiceID(service.ID())
	for _, am := range monitors {
		am.removeServiceFromMonitor(service)

		if len(am.serviceSet) == 0 {
			log.WithFields(log.Fields{
				"monitorType":   am.config.CoreConfig().Type,
				"discoveryRule": am.config.CoreConfig().DiscoveryRule,
			}).Info("Monitor no longer has active services, shutting down")

			am.doomed = true
		}
	}
	mm.deleteDoomedMonitors()

	log.WithFields(log.Fields{
		"service": service,
	}).Info("No longer monitoring service")
}

func (mm *MonitorManager) isServiceMonitored(service services.Endpoint) bool {
	monitors := mm.monitorsForServiceID(service.ID())
	return len(monitors) > 0
}

// Shutdown will shutdown all managed monitors and deinitialize the manager.
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
