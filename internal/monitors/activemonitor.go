package monitors

import (
	"reflect"

	"github.com/creasty/defaults"
	"github.com/pkg/errors"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/core/meta"
	"github.com/signalfx/signalfx-agent/internal/core/services"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

// ActiveMonitor is a wrapper for an actual monitor instance that keeps some
// metadata about the monitor, such as the set of service endpoints attached to
// the monitor, as well as a copy of its configuration.  It exposes a lot of
// methods to help manage the monitor as well.
type ActiveMonitor struct {
	instance   interface{}
	id         types.MonitorID
	configHash uint64
	agentMeta  *meta.AgentMeta
	output     types.Output
	config     config.MonitorCustomConfig
	endpoint   services.Endpoint
	// Is the monitor marked for deletion?
	doomed bool
}

// Does some reflection magic to pass the right type to the Configure method of
// each monitor
func (am *ActiveMonitor) configureMonitor(monConfig config.MonitorCustomConfig) error {
	monConfig = utils.CloneInterface(monConfig).(config.MonitorCustomConfig)
	if err := defaults.Set(monConfig); err != nil {
		return err
	}

	if am.endpoint != nil {
		err := config.DecodeExtraConfig(am.endpoint, monConfig, false)
		if err != nil {
			return errors.Wrap(err, "Could not inject endpoint config into monitor config")
		}
	}

	am.config = monConfig
	am.config.MonitorConfigCore().MonitorID = am.id

	if err := validateConfig(monConfig); err != nil {
		return err
	}

	am.injectAgentMetaIfNeeded()
	am.injectOutputIfNeeded()

	return config.CallConfigure(am.instance, monConfig)
}

func (am *ActiveMonitor) endpointID() services.ID {
	if am.endpoint == nil {
		return ""
	}
	return am.endpoint.Core().ID
}

func (am *ActiveMonitor) injectOutputIfNeeded() bool {
	outputValue := utils.FindFieldWithEmbeddedStructs(am.instance, "Output",
		reflect.TypeOf((*types.Output)(nil)).Elem())

	if !outputValue.IsValid() {
		return false
	}

	outputValue.Set(reflect.ValueOf(am.output))

	return true
}

// Sets the `AgentMeta` field on a monitor if it is present to the agent
// metadata service. Returns whether the field was actually set.
// N.B. that the values in AgentMeta are subject to change at any time.  There
// is no notification mechanism for changes, so a monitor should pull the value
// from the struct each time it needs it and not cache it.
func (am *ActiveMonitor) injectAgentMetaIfNeeded() bool {
	agentMetaValue := utils.FindFieldWithEmbeddedStructs(am.instance, "AgentMeta",
		reflect.TypeOf(&meta.AgentMeta{}))

	if !agentMetaValue.IsValid() {
		return false
	}

	agentMetaValue.Set(reflect.ValueOf(am.agentMeta))

	return true
}

// Shutdown calls Shutdown on the monitor instance if it is provided.
func (am *ActiveMonitor) Shutdown() {
	if sh, ok := am.instance.(Shutdownable); ok {
		sh.Shutdown()
	}
}
