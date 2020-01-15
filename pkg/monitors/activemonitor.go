package monitors

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"go.uber.org/atomic"

	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
	"github.com/signalfx/defaults"

	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/core/meta"
	"github.com/signalfx/signalfx-agent/pkg/core/services"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/utils"
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
	output     types.FilteringOutput
	config     config.MonitorCustomConfig
	endpoint   services.Endpoint
	// cancel function for the parent context if it is a Collectable instance
	cancel context.CancelFunc
	// Is the monitor marked for deletion?
	doomed bool

	collectFailures  atomic.Uint64
	collectCalls     atomic.Uint64
	intervalExceeded atomic.Uint64
}

func renderConfig(monConfig config.MonitorCustomConfig, endpoint services.Endpoint) (config.MonitorCustomConfig, error) {
	monConfig = utils.CloneInterface(monConfig).(config.MonitorCustomConfig)
	if err := defaults.Set(monConfig); err != nil {
		return nil, err
	}

	if endpoint != nil {
		err := config.DecodeExtraConfig(endpoint, monConfig, false)
		if err != nil {
			return nil, errors.Wrap(err, "Could not inject endpoint config into monitor config")
		}

		for configKey, rule := range monConfig.MonitorConfigCore().ConfigEndpointMappings {
			cem := &services.ConfigEndpointMapping{
				Endpoint:  endpoint,
				ConfigKey: configKey,
				Rule:      rule,
			}
			if err := config.DecodeExtraConfig(cem, monConfig, false); err != nil {
				return nil, fmt.Errorf("could not process config mapping: %s => %s -- %s", configKey, rule, err.Error())
			}
		}
	}

	// Wipe out the other config that has already been decoded since it is not
	// redundant.
	monConfig.MonitorConfigCore().OtherConfig = nil
	return monConfig, nil
}

// Does some reflection magic to pass the right type to the Configure method of
// each monitor
func (am *ActiveMonitor) configureMonitor(monConfig config.MonitorCustomConfig) error {
	monConfig.MonitorConfigCore().MonitorID = am.id
	for k, v := range monConfig.MonitorConfigCore().ExtraDimensions {
		am.output.AddExtraDimension(k, v)
	}

	for k, v := range monConfig.MonitorConfigCore().ExtraDimensionsFromEndpoint {
		val, err := services.EvaluateRule(am.endpoint, v, true, true)
		if err != nil {
			return err
		}
		am.output.AddExtraDimension(k, fmt.Sprintf("%v", val))
	}

	if err := validateConfig(monConfig); err != nil {
		return err
	}

	am.config = monConfig
	am.injectAgentMetaIfNeeded()
	am.injectOutputIfNeeded()

	if err := config.CallConfigure(am.instance, monConfig); err != nil {
		return err
	}

	if mon, ok := am.instance.(Collectable); ok {
		var ctx context.Context
		ctx, am.cancel = context.WithCancel(context.Background())
		interval := time.Duration(am.config.MonitorConfigCore().IntervalSeconds) * time.Second

		// TODO: Would be good to track lingering monitors where the monitor stops
		// but the goroutine that called Collect is still running due to being blocked.
		// Could possibly use a Deadline context as well to cancel after some unusually
		// long time.
		utils.RunOnInterval(ctx, func() {
			// TODO: Maybe put this on am instance instead.
			logger := logrus.WithFields(logrus.Fields{"monitorType": monConfig.MonitorConfigCore().Type})

			start := time.Now()
			if err := mon.Collect(ctx); err != nil {
				am.collectFailures.Inc()
				logger.Errorf("collecting data from monitor failed: %s", err)
			}
			am.collectCalls.Inc()
			elapsed := time.Since(start)

			if elapsed > interval {
				am.intervalExceeded.Inc()
				logger.Warnf("monitor %s took too long to run (%s) which will cause lagging datapoints", am.id, elapsed)
			}
		}, interval)
	}

	return nil
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
		// Try and find FilteringOutput type
		outputValue = utils.FindFieldWithEmbeddedStructs(am.instance, "Output",
			reflect.TypeOf((*types.FilteringOutput)(nil)).Elem())
		if !outputValue.IsValid() {
			return false
		}
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
	if am.cancel != nil {
		am.cancel()
	}

	if sh, ok := am.instance.(Shutdownable); ok {
		sh.Shutdown()
	}
}
