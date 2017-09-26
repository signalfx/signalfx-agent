package monitors

import (
	"reflect"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/core/writer"
	log "github.com/sirupsen/logrus"
)

// ActiveMonitor is a wrapper for an actual monitor instance that keeps some
// metadata about the monitor, such as the set of service endpoints attached to
// the monitor, as well as a copy of its configuration.  It exposes a lot of
// methods to help manage the monitor as well.
type ActiveMonitor struct {
	instance   interface{}
	config     config.MonitorCustomConfig
	serviceSet map[services.ID]services.Endpoint
	// Is the monitor marked for deletion?
	doomed bool
}

// Does some reflection magic to pass the right type to the Configure method of
// each monitor
func (am *ActiveMonitor) configureMonitor(monConfig config.MonitorCustomConfig) bool {
	am.config = monConfig

	if !config.CallConfigure(am.instance, monConfig) {
		return false
	}

	return am.injectAndRemoveManualServices()
}

// Add new services and remove old ones that are no longer configured
func (am *ActiveMonitor) injectAndRemoveManualServices() bool {
	ses := config.ServiceEndpointsFromConfig(am.config)
	if len(ses) > 0 {
		for k := range am.serviceSet {
			am.removeServiceFromMonitor(am.serviceSet[k])
		}

		for i := range ses {
			if !am.injectServiceToMonitorInstance(ses[i]) {
				return false
			}
		}
	}

	return true
}

func (am *ActiveMonitor) injectServiceToMonitorInstance(service services.Endpoint) bool {
	if inst, ok := am.instance.(InjectableMonitor); ok {
		// Make sure this is done before injecting service to monitor!
		service.AddMatchingMonitor(am.config.CoreConfig().ID)

		inst.AddService(service)
		am.serviceSet[service.ID()] = service

		return true
	}

	log.WithFields(log.Fields{
		"monitorType": am.config.CoreConfig().Type,
	}).Error("Monitor does not provide the service injection methods!")
	return false
}

func (am *ActiveMonitor) removeServiceFromMonitor(service services.Endpoint) bool {
	if inst, ok := am.instance.(InjectableMonitor); ok {
		// Make sure this is done before removing service from monitor!
		service.RemoveMatchingMonitor(am.config.CoreConfig().ID)
		inst.RemoveService(service)
		delete(am.serviceSet, service.ID())

		return true
	}

	log.WithFields(log.Fields{
		"service": service,
	}).Error("Monitor does not provide the service injection methods!")
	return false
}

// Sets the `DPs` field on a monitor if it is present to the datapoint channel.
// Returns whether the field was actually set.
func (am *ActiveMonitor) injectDatapointChannelIfNeeded(dpChan chan<- *datapoint.Datapoint) bool {
	instanceValue := reflect.Indirect(reflect.ValueOf(am.instance))
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
	dpsValue.Set(reflect.ValueOf(dpChan))

	return true
}

// Sets the `Events` field on a monitor if it is present to the events channel.
// Returns whether the field was actually set.
func (am *ActiveMonitor) injectEventChannelIfNeeded(eventChan chan<- *event.Event) bool {
	instanceValue := reflect.Indirect(reflect.ValueOf(am.instance))
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
	eventsValue.Set(reflect.ValueOf(eventChan))

	return true
}

// Sets the `DimProps` field on a monitor if it is present to the dimension
// properties channel. Returns whether the field was actually set.
func (am *ActiveMonitor) injectDimPropertiesChannelIfNeeded(dimPropChan chan<- *writer.DimProperties) bool {
	instanceValue := reflect.Indirect(reflect.ValueOf(am.instance))
	dpsValue := instanceValue.FieldByName("DimProps")
	if !dpsValue.IsValid() {
		return false
	}
	if dpsValue.Type() != reflect.ChanOf(reflect.SendDir, reflect.TypeOf(&writer.DimProperties{})) {
		log.WithFields(log.Fields{
			"pkgPath": instanceValue.Type().PkgPath(),
		}).Error("Monitor instance has 'DimProps' member but is not of type 'chan<- *writer.DimProperties'")
		return false
	}
	dpsValue.Set(reflect.ValueOf(dimPropChan))

	return true
}

// Shutdown calls Shutdown on the monitor instance if it is provided.
func (am *ActiveMonitor) Shutdown() {
	if sh, ok := am.instance.(Shutdownable); ok {
		sh.Shutdown()
	}
}
