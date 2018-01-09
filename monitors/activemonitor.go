package monitors

import (
	"fmt"
	"reflect"

	"github.com/creasty/defaults"
	"github.com/pkg/errors"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/core/writer"
	"github.com/signalfx/neo-agent/utils"
)

// ActiveMonitor is a wrapper for an actual monitor instance that keeps some
// metadata about the monitor, such as the set of service endpoints attached to
// the monitor, as well as a copy of its configuration.  It exposes a lot of
// methods to help manage the monitor as well.
type ActiveMonitor struct {
	instance interface{}
	id       MonitorID
	config   config.MonitorCustomConfig
	endpoint services.Endpoint
	// Is the monitor marked for deletion?
	doomed bool
}

// Does some reflection magic to pass the right type to the Configure method of
// each monitor
func (am *ActiveMonitor) configureMonitor(monConfig config.MonitorCustomConfig) error {
	monConfig = utils.CloneInterface(monConfig).(config.MonitorCustomConfig)

	if err := defaults.Set(monConfig.CoreConfig()); err != nil {
		// This is only caused by a programming bug, not bad user input
		panic(fmt.Sprintf("Config defaults are wrong types: %s", err))
	}

	if am.endpoint != nil {
		err := config.DecodeExtraConfigStrict(am.endpoint, monConfig)
		if err != nil {
			return errors.Wrap(err, "Could not inject endpoint config into monitor config")
		}
		for k, v := range am.endpoint.Dimensions() {
			monConfig.CoreConfig().ExtraDimensions[k] = v
		}
	}

	am.config = monConfig

	if err := validateFields(monConfig); err != nil {
		return err
	}
	return config.CallConfigure(am.instance, monConfig)
}

// Sets the `DPs` field on a monitor if it is present to the datapoint channel.
// Returns whether the field was actually set.
func (am *ActiveMonitor) injectDatapointChannelIfNeeded(dpChan chan<- *datapoint.Datapoint) bool {
	dpsValue := utils.FindFieldWithEmbeddedStructs(am.instance, "DPs",
		reflect.ChanOf(reflect.SendDir, reflect.TypeOf(&datapoint.Datapoint{})))

	if !dpsValue.IsValid() {
		return false
	}

	dpsValue.Set(reflect.ValueOf(dpChan))
	return true
}

func (am *ActiveMonitor) endpointID() services.ID {
	if am.endpoint == nil {
		return ""
	}
	return am.endpoint.Core().ID
}

// Sets the `Events` field on a monitor if it is present to the events channel.
// Returns whether the field was actually set.
func (am *ActiveMonitor) injectEventChannelIfNeeded(eventChan chan<- *event.Event) bool {
	eventsValue := utils.FindFieldWithEmbeddedStructs(am.instance, "Events",
		reflect.ChanOf(reflect.SendDir, reflect.TypeOf(&event.Event{})))

	if !eventsValue.IsValid() {
		return false
	}

	eventsValue.Set(reflect.ValueOf(eventChan))

	return true
}

// Sets the `DimProps` field on a monitor if it is present to the dimension
// properties channel. Returns whether the field was actually set.
func (am *ActiveMonitor) injectDimPropertiesChannelIfNeeded(dimPropChan chan<- *writer.DimProperties) bool {
	dimPropsValue := utils.FindFieldWithEmbeddedStructs(am.instance, "DimProps",
		reflect.ChanOf(reflect.SendDir, reflect.TypeOf(&writer.DimProperties{})))

	if !dimPropsValue.IsValid() {
		return false
	}

	dimPropsValue.Set(reflect.ValueOf(dimPropChan))

	return true
}

// Shutdown calls Shutdown on the monitor instance if it is provided.
func (am *ActiveMonitor) Shutdown() {
	if sh, ok := am.instance.(Shutdownable); ok {
		sh.Shutdown()
	}
}
