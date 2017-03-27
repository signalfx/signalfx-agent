package services

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"sync"

	"fmt"

	ruler "github.com/hopkinsth/go-ruler"
	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/services"
	"github.com/spf13/viper"
)

const (
	pluginType = "filters/service-rules"
)

// DiscoveryRuleset that names a set of service discovery rules
type DiscoveryRuleset struct {
	Config string
	Type string
	Enabled bool
	Labels []string
	// Rules are criteria for service identification
	Rules []struct {
		Comparator string
		Path       string
		Value      interface{}
	}
}

// DiscoverySignatures with name
type DiscoverySignatures struct {
	Name       string
	Signatures []DiscoveryRuleset
}

// RuleFilter filters instances based on rules
type RuleFilter struct {
	plugins.Plugin
	mutex        sync.Mutex
	serviceRules []*DiscoverySignatures
}

func init() {
	plugins.Register(pluginType, NewRuleFilter)
}

// NewRuleFilter creates a new instance
func NewRuleFilter(name string, config *viper.Viper) (plugins.IPlugin, error) {
	plugin, err := plugins.NewPlugin(name, pluginType, config)
	if err != nil {
		return nil, err
	}

	filter := &RuleFilter{Plugin: plugin}
	if err := filter.load(); err != nil {
		// Don't return an error as user may fix configuration and reload will fix it.
		log.Printf("failed to load initial service files: %s", err)
	}

	return filter, nil
}

// GetWatchFiles returns list of files that when changed will trigger reload.
func (filter *RuleFilter) GetWatchFiles(config *viper.Viper) []string {
	return config.GetStringSlice("servicesfiles")
}

// loadServiceSignatures reads discovery rules from file
func loadServiceSignatures(servicesFiles []string) ([]*DiscoverySignatures, error) {
	var serviceRules []*DiscoverySignatures

	for _, servicesFile := range servicesFiles {
		log.Printf("loading service discovery signatures from %s", servicesFile)

		var signatures *DiscoverySignatures
		jsonContent, err := ioutil.ReadFile(servicesFile)

		if err != nil {
			return serviceRules, fmt.Errorf("reading %s failed: %s", servicesFile, err)
		}

		if err := json.Unmarshal(jsonContent, &signatures); err != nil {
			return serviceRules, fmt.Errorf("unmarshaling %s failed: %s", servicesFile, err)
		}

		serviceRules = append(serviceRules, signatures)
	}
	return serviceRules, nil
}

// Matches if service instance satisfies rules
func matches(si *services.Instance, ruleset DiscoveryRuleset) (bool, error) {
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
		"NetworkPublicPort":  float64(si.Port.PublicPort),
		"NetworkPrivatePort": float64(si.Port.PrivatePort),
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
func (filter *RuleFilter) Map(sis services.Instances) (services.Instances, error) {
	filter.mutex.Lock()
	defer filter.mutex.Unlock()

	applicableServices := make(services.Instances, 0, len(sis))

	// Find the first rule that matches each service instance.
OUTER:
	for i := range sis {
		for _, signature := range filter.serviceRules {
			for _, ruleset := range signature.Signatures {
				if ruleset.Enabled {
					matches, err := matches(&sis[i], ruleset)
					if err != nil {
						return nil, err
					}

					if matches {
						// add as service to monitor
						// FIXME: what if it's not a known service type?
						sis[i].Service.Type = services.ServiceType(ruleset.Type)
						sis[i].Config = ruleset.Config
						for _, label := range ruleset.Labels {
							if val, ok := sis[i].Container.Labels[label]; ok {
								sis[i].Orchestration.Dims[label] = val
							}
						}
						applicableServices = append(applicableServices, sis[i])
						// Rule found, continue to next service instance.
						continue OUTER
					}
				}
			}
		}
	}

	return applicableServices, nil
}

// load the filter rules
func (filter *RuleFilter) load() error {
	files := filter.Config.GetStringSlice("servicesfiles")
	serviceRules, err := loadServiceSignatures(files)

	if err != nil {
		return err
	}

	filter.serviceRules = serviceRules
	return nil
}

// Reload plugin
func (filter *RuleFilter) Reload(config *viper.Viper) error {
	filter.mutex.Lock()
	defer filter.mutex.Unlock()

	filter.Config = config
	return filter.load()
}
