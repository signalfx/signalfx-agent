package monitors

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	log "github.com/sirupsen/logrus"
)

// Used to validate configuration that is common to all monitors up front
func validateCommonConfig(conf *config.MonitorConfig) bool {
	result := true

	// Validate discovery rules
	if conf.DiscoveryRule != "" {
		if !services.ValidateDiscoveryRule(conf.DiscoveryRule) {
			log.WithFields(log.Fields{
				"monitorType": conf.Type,
			}).Error("Could not validate discovery rule for monitor")

			result = false
		}

		manualEndpoints := conf.OtherConfig["serviceEndpoints"]
		if manualEndpoints != nil && len(manualEndpoints.([]interface{})) > 0 {
			log.WithFields(log.Fields{
				"monitorType": conf.Type,
			}).Error("Cannot have a monitor with discoveryRule and serviceEndpoints.  " +
				"Please split your config into two separate monitors.")

			result = false
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

// ValidateEndpointConfig accepts both the common configuration as well as the
// service endpoints defined for a service and validates that any requiredFields
// are present.
func ValidateEndpointConfig(common services.Endpoint) bool {

	return true
}
