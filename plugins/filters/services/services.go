package services

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"strconv"

	ruler "github.com/hopkinsth/go-ruler"
	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/services"
	"github.com/spf13/viper"
)

// ServiceDiscoveryRuleset that names a set of service discovery rules
type ServiceDiscoveryRuleset struct {
	Name string
	Type string
	// Rules are criteria for service identification
	Rules []struct {
		Comparator string
		Path       string
		Value      interface{}
	}
}

// ServiceDiscoverySignatures with name
type ServiceDiscoverySignatures struct {
	Name       string
	Signatures []ServiceDiscoveryRuleset
}

// RuleFilter filters instances based on rules
type RuleFilter struct {
	plugins.Plugin
	serviceRules []*ServiceDiscoverySignatures
}

// NewRuleFilter creates a new instance
func NewRuleFilter(name string, config *viper.Viper) (*RuleFilter, error) {
	var (
		signatures    []*ServiceDiscoverySignatures
		servicesFiles []string
		err           error
	)

	plugin, err := plugins.NewPlugin(name, config)
	if err != nil {
		return nil, err
	}

	if servicesFiles = plugin.Config.GetStringSlice("servicesfiles"); len(servicesFiles) == 0 {
		return nil, errors.New("servicesFiles configuration value missing")
	}

	for _, servicesFile := range servicesFiles {
		log.Printf("loading service discovery signatures from %s", servicesFile)
		loaded, err := loadServiceSignatures(servicesFile)
		if err != nil {
			return nil, err
		}
		signatures = append(signatures, loaded)
	}

	return &RuleFilter{plugin, signatures}, nil
}

// loadServiceSignatures reads discovery rules from file
func loadServiceSignatures(servicesFile string) (*ServiceDiscoverySignatures, error) {
	var signatures ServiceDiscoverySignatures
	jsonContent, err := ioutil.ReadFile(servicesFile)
	if err != nil {
		return &signatures, err
	}

	if err := json.Unmarshal(jsonContent, &signatures); err != nil {
		return &signatures, err
	}
	return &signatures, nil
}

// Matches if service instance satisfies rules
func matches(si *services.ServiceInstance, ruleset ServiceDiscoveryRuleset) (bool, error) {
	jsonRules, err := json.Marshal(ruleset.Rules)
	if err != nil {
		return false, err
	}

	engine, err := ruler.NewRulerWithJson(jsonRules)
	if err != nil {
		return false, err
	}

	sm := map[string]interface{}{
		"ContainerID":        si.Container.ID,
		"ContainerName":      si.Container.Names[0],
		"ContainerImage":     si.Container.Image,
		"ContainerPod":       si.Container.Pod,
		"ContainerCommand":   si.Container.Command,
		"ContainerState":     si.Container.State,
		"NetworkIP":          si.Port.IP,
		"NetworkType":        si.Port.Type,
		"NetworkPublicPort":  strconv.FormatUint(uint64(si.Port.PublicPort), 10),
		"NetworkPrivatePort": strconv.FormatUint(uint64(si.Port.PrivatePort), 10),
	}

	for key, val := range si.Container.Labels {
		sm["ContainerLabel-"+key] = val
	}

	for key, val := range si.Port.Labels {
		sm["NetworkLabel-"+key] = val
	}

	return engine.Test(sm), nil
}

// Map matches discovered service instances to a plugin type.
func (filter *RuleFilter) Map(sis services.ServiceInstances) (services.ServiceInstances, error) {
	applicableServices := make(services.ServiceInstances, 0, len(sis))

	// Find the first rule that matches each service instance.
OUTER:
	for i := range sis {
		for _, signature := range filter.serviceRules {
			for _, ruleset := range signature.Signatures {
				matches, err := matches(&sis[i], ruleset)
				if err != nil {
					return nil, err
				}

				if matches {
					// set service name to ruleset name and add as service to monitor
					sis[i].Service.Name = ruleset.Name
					// FIXME: what if it's not a known service type?
					sis[i].Service.Type = services.ServiceType(ruleset.Type)
					applicableServices = append(applicableServices, sis[i])
					// Rule found, continue to next service instance.
					continue OUTER
				}
			}
		}
	}

	return applicableServices, nil
}
