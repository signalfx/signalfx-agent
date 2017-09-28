package monitors

import (
	"errors"
	"fmt"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
)

// Used to validate configuration that is common to all monitors up front
func validateConfig(monConfig config.MonitorCustomConfig) error {
	conf := monConfig.CoreConfig()

	if _, ok := MonitorFactories[conf.Type]; !ok {
		return errors.New("Monitor type not recognized")
	}

	manualEndpoints := conf.OtherConfig["serviceEndpoints"]
	hasManualEndpoints := manualEndpoints != nil && len(manualEndpoints.([]interface{})) > 0

	inst := newMonitor(conf.Type)
	_, takesServices := inst.(InjectableMonitor)
	if takesServices && conf.DiscoveryRule == "" && !hasManualEndpoints {
		return fmt.Errorf("Monitor %s takes services but did not specify any discovery rule or manually defined services", conf.Type)
	}

	// Validate discovery rules
	if conf.DiscoveryRule != "" {
		err := services.ValidateDiscoveryRule(conf.DiscoveryRule)
		if err != nil {
			return errors.New("Could not validate discovery rule: " + err.Error())
		}

		if hasManualEndpoints {
			return errors.New("Cannot have a monitor with discoveryRule and serviceEndpoints. " +
				"Please split your config into two separate monitors.")
		}
	}

	return config.ValidateCustomConfig(monConfig)
}
