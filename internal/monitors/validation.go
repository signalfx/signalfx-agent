package monitors

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/core/config/validation"
	"github.com/signalfx/signalfx-agent/internal/core/services"
)

// Used to validate configuration that is common to all monitors up front.
// allowSuppliedFields will allow host/port and configEndpointMappings fields
// to fail validation if they aren't present.
func validateConfig(monConfig config.MonitorCustomConfig, allowSuppliedFields bool) error {
	conf := monConfig.MonitorConfigCore()

	if _, ok := MonitorFactories[conf.Type]; !ok {
		return errors.New("monitor type not recognized")
	}

	if conf.IntervalSeconds <= 0 {
		return fmt.Errorf("invalid intervalSeconds provided: %d", conf.IntervalSeconds)
	}

	takesEndpoints := configAcceptsEndpoints(monConfig)
	if !takesEndpoints && conf.DiscoveryRule != "" {
		return fmt.Errorf("monitor %s does not support discovery but has a discovery rule", conf.Type)
	}

	// Validate discovery rules
	if conf.HasAutoDiscovery() {
		err := services.ValidateDiscoveryRule(conf.DiscoveryRule)
		if err != nil {
			return errors.New("discovery rule is invalid: " + err.Error())
		}
	}

	if len(conf.ConfigEndpointMappings) > 0 && len(conf.DiscoveryRule) == 0 {
		return errors.New("configEndpointMappings is not useful without a discovery rule")
	}

	if err := validation.ValidateStruct(monConfig); err != nil {
		if !allowSuppliedFields || !errorMightBeAcceptable(err, conf) {
			return err
		}
	}

	if err := validation.ValidateCustomConfig(monConfig); err != nil {
		if !allowSuppliedFields || !errorMightBeAcceptable(err, conf) {
			return err
		}
	}

	return nil
}

// Also prunes the FieldErrors field on the passed in err if applicable
func errorMightBeAcceptable(origErr error, conf *config.MonitorConfig) bool {
	outIdx := 0
	if err, ok := origErr.(*validation.StructError); ok {
		acceptable := true
		for _, fe := range err.FieldErrors {
			if conf.HasAutoDiscovery() && (fe.Field == "host" || fe.Field == "port") {
				continue
			} else if _, ok := conf.ConfigEndpointMappings[fe.Field]; ok {
				// Config might be supplied by the endpoint metadata so let it through,
				// it should be validated later again before the monitor is actually
				// instantiated.
				continue
			} else {
				acceptable = false
				// Only keep field errors that aren't acceptable
				err.FieldErrors[outIdx] = fe
				outIdx++
			}
		}
		err.FieldErrors = err.FieldErrors[:outIdx]
		return acceptable
	}
	return false
}

func configAcceptsEndpoints(monConfig config.MonitorCustomConfig) bool {
	confVal := reflect.Indirect(reflect.ValueOf(monConfig))
	coreConfField, ok := confVal.Type().FieldByName("MonitorConfig")
	if !ok {
		return false
	}
	return coreConfField.Tag.Get("acceptsEndpoints") == "true"
}
