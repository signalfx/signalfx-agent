package monitors

import (
	"errors"
	"fmt"
	"reflect"

	validator "gopkg.in/go-playground/validator.v9"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
)

// Used to validate configuration that is common to all monitors up front
func validateConfig(monConfig config.MonitorCustomConfig) error {
	conf := monConfig.MonitorConfigCore()

	if _, ok := MonitorFactories[conf.Type]; !ok {
		return errors.New("Monitor type not recognized")
	}

	takesEndpoints := configAcceptsEndpoints(monConfig)
	if !takesEndpoints && conf.DiscoveryRule != "" {
		return fmt.Errorf("Monitor %s does not support discovery but has a discovery rule", conf.Type)
	}

	// Validate discovery rules
	if conf.DiscoveryRule != "" {
		err := services.ValidateDiscoveryRule(conf.DiscoveryRule)
		if err != nil {
			return errors.New("Could not validate discovery rule: " + err.Error())
		}
	}

	validate := validator.New()
	err := validate.Struct(monConfig)
	if err != nil {
		return err
	}

	return config.ValidateCustomConfig(monConfig)
}

func configAcceptsEndpoints(monConfig config.MonitorCustomConfig) bool {
	confVal := reflect.Indirect(reflect.ValueOf(monConfig))
	coreConfField, ok := confVal.Type().FieldByName("MonitorConfig")
	if !ok {
		return false
	}
	return coreConfField.Tag.Get("acceptsEndpoints") == "true"
}

func isConfigUnique(conf *config.MonitorConfig, otherConfs []config.MonitorConfig) bool {
	for _, c := range otherConfs {
		if c.MonitorConfigCore().Equals(conf) {
			return true
		}
	}
	return false
}
