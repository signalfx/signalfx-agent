package monitors

import (
	"github.com/signalfx/neo-agent/core/config"
	log "github.com/sirupsen/logrus"
)

// Used to validate configuration that is common to all monitors up front
func validateCommonConfig(conf *config.MonitorConfig) bool {
	result := true
	// Validate discovery rules
	if conf.DiscoveryRule != "" {
		expr, err := parseRuleText(conf.DiscoveryRule)
		if err != nil {
			log.WithFields(log.Fields{
				"rule":        conf.DiscoveryRule,
				"monitorType": conf.Type,
			}).Error("Syntax error in discovery rule")

			result = false
		}

		variables := expr.Vars()
		for _, v := range variables {
			if !validRuleIdentifiers[v] {
				log.WithFields(log.Fields{
					"rule":        conf.DiscoveryRule,
					"monitorType": conf.Type,
					"variable":    v,
				}).Error("Unknown variable in discovery rule")

				result = false
			}
		}
	}

	if _, ok := MonitorFactories[conf.Type]; !ok {
		log.WithFields(log.Fields{
			"monitorType": conf.Type,
		}).Error("Monitor type not recognized")

		result = false
	}

	return result
}

type Validatable interface {
	Validate() bool
}

// validate monitor-specific config ahead of time for a specific monitor
// configuration.  This way, the Configure method of monitors will be
// guaranteed to receive valid configuration.  The monitor-specific
// configuration struct must implement the Validate method that returns a bool.
func validateCustomConfig(conf interface{}) bool {
	if v, ok := conf.(Validatable); ok {
		return v.Validate()
	}
	return true
}
