package integrations

import (
	"errors"
	"log"
	"reflect"
	"sync"

	"fmt"

	"os"

	"strings"

	"github.com/Knetic/govaluate"
	"github.com/docker/libkv/store"
	"github.com/signalfx/neo-agent/config"
	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/services"
	"github.com/signalfx/neo-agent/utils"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

const (
	pluginType = "filters/integrations"
)

var functions = map[string]govaluate.ExpressionFunction{
	"Get": func(args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, errors.New("Get takes 2 args")
		}
		arg1 := args[0]
		arg2 := args[1]

		labelMap, ok := arg1.(map[string]string)
		if !ok {
			return nil, errors.New("label must be a map[string]string")
		}
		label, ok := arg2.(string)
		if !ok {
			return nil, errors.New("label must be of type string")
		}

		if val, ok := labelMap[label]; ok {
			return val, nil
		}

		return nil, nil
	},
	"Contains": func(args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, errors.New("Contains takes 2 args")
		}
		arg1 := args[0]
		arg2 := args[1]

		labelMap, ok := arg1.(map[string]string)
		if !ok {
			return nil, errors.New("label must be a map[string]string")
		}
		label, ok := arg2.(string)
		if !ok {
			return nil, errors.New("label must be of type string")
		}

		if _, ok := labelMap[label]; ok {
			return true, nil
		}

		return false, nil
	},
}

type integConfig struct {
	Rule     string
	Template string
	Vars     map[string]interface{}
	Labels   *[]string
}

type configFile struct {
	Disabled     []string
	Integrations map[services.ServiceType]*struct {
		Plugin         string
		Rule           string
		Template       string
		Vars           map[string]interface{}
		Labels         *[]string
		Configurations []*integConfig
	}
}

type configuration struct {
	serviceType services.ServiceType
	ruleText    string
	rule        *govaluate.EvaluableExpression
	vars        map[string]interface{}
	template    string
	// plugin is the (currently collectd) plugin to be used
	plugin string
	labels []string
}

// Filter filters instances based on rules and maps service configuration
type Filter struct {
	configurations []*configuration
	// builtins       configs
	// overrides      configs
	assets *config.AssetSyncer
	mutex  sync.Mutex
}

func init() {
	plugins.Register(pluginType, func() interface{} {
		f := &Filter{assets: config.NewAssetSyncer()}
		f.assets.Start(f.onAssetChange)
		return f
	})
}

// readConfigs reads *.yml from key pairs into a an array of byte arrays
func readConfigs(configs []*store.KVPair) ([][]byte, error) {
	var configFiles [][]byte

	for _, pair := range configs {
		if !strings.HasSuffix(pair.Key, ".yml") {
			continue
		}
		configFiles = append(configFiles, pair.Value)
	}

	return configFiles, nil
}

func loadConfigs(configs [][]byte) ([]configFile, error) {
	var configFiles []configFile

	for _, config := range configs {
		var c configFile
		if err := yaml.Unmarshal(config, &c); err != nil {
			return nil, err
		}
		configFiles = append(configFiles, c)
	}

	return configFiles, nil
}

func loadBuiltins(builtins []configFile) (map[services.ServiceType]*configuration, error) {
	builtinsMap := map[services.ServiceType]*configuration{}
	for _, builtin := range builtins {
		for integName, integ := range builtin.Integrations {
			var rule *govaluate.EvaluableExpression
			var err error
			var labels []string

			if len(integ.Configurations) != 0 {
				return nil, fmt.Errorf("found unexpected configuration in builtin %s", integName)
			}

			if _, ok := builtinsMap[integName]; ok {
				return nil, fmt.Errorf("found existing builtin for %s", integName)
			}

			if integ.Rule != "" {
				rule, err = govaluate.NewEvaluableExpressionWithFunctions(integ.Rule, functions)
				if err != nil {
					return nil, fmt.Errorf("error constructing rule for %s: %s", integName, err)
				}
			}

			// If an integration specifies a specific plugin name use that.
			// Otherwise use the integration name/type. This is used for JMX
			// integrations that are separate integrations but all get grouped
			// by a jmx plugin type.
			plugin := integ.Plugin
			if plugin == "" {
				plugin = string(integName)
			}

			if integ.Labels != nil {
				labels = *integ.Labels
			}

			builtinsMap[integName] = &configuration{integName, integ.Rule, rule, integ.Vars, integ.Template,
				plugin, labels}
		}
	}

	return builtinsMap, nil
}

// resolveVars resolves any environment variable references into their values
func resolveVars(vars map[string]interface{}) {
	for k, v := range vars {
		if entry, ok := v.(map[interface{}]interface{}); ok {
			if envVariable, ok := entry["env"]; ok {
				if strVar, ok := envVariable.(string); ok {
					vars[k] = os.Getenv(strVar)
				}
			}
		}
	}
}

// buildConfigurations takes an array of builtins and overrides and merges them
// together to form a list of configurations/rules that service instances are
// applied to
func buildConfigurations(builtins, overrides []configFile) ([]*configuration, error) {
	var configs []*configuration

	builtinsMap, err := loadBuiltins(builtins)
	if err != nil {
		return nil, err
	}

	configured := map[services.ServiceType]bool{}

	for _, config := range overrides {
		for integName, integ := range config.Integrations {
			configured[integName] = true
			builtinInteg := builtinsMap[integName]
			if builtinInteg == nil {
				return nil, fmt.Errorf("%s is missing builtin", integName)
			}

			if len(integ.Configurations) == 0 {
				var template, ruleText string
				var labels []string

				if ruleText = utils.FirstNonEmpty(integ.Rule, builtinInteg.ruleText); ruleText == "" {
					return nil, fmt.Errorf("rule is required for integration %s", integName)
				}

				rule, err := govaluate.NewEvaluableExpressionWithFunctions(ruleText, functions)
				if err != nil {
					return nil, fmt.Errorf("error constructing rule for %s: %s", integName, err)
				}

				if template = utils.FirstNonEmpty(integ.Template, builtinInteg.template); template == "" {
					return nil, fmt.Errorf("integration %s must have a configured template", integName)
				}

				vars := utils.MergeMaps(builtinInteg.vars, integ.Vars)
				resolveVars(vars)

				// If an integration specifies a specific plugin name use that.
				// Otherwise use the integration name/type. This is used for JMX
				// integrations that are separate integrations but all get grouped
				// by a jmx plugin type.
				plugin := integ.Plugin
				if plugin == "" {
					plugin = string(integName)
				}

				if integ.Labels != nil {
					labels = append(labels, *integ.Labels...)
				} else {
					labels = builtinInteg.labels
				}

				// TODO: check that it's a supported service
				configs = append(configs, &configuration{services.ServiceType(integName), ruleText, rule,
					vars, template, plugin, labels})
			} else {
				for configName, config := range integ.Configurations {
					var template, ruleText string
					var labels []string

					// Rule merging. If the configuration doesn't specify a rule
					// and the integration it's a part of doesn't specify a rule
					// default to the builtin rule. Otherwise if both the
					// configuration and an integration it's a part of specify a
					// rule AND them together. Otherwise default to the
					// configuration's rule. If the user specifies a
					// configuration rule we don't try to merge it with the
					// builtin rule because the user has no way to
					// disable/override that merging.

					// TODO: let only 1 configuration have an empty rule (the
					// "default") it will match everything at the integration
					// level LAST. Does it need to AND with builtin? Or maybe
					// configurations should just have an explicit order
					// instead.
					if config.Rule == "" {
						return nil, fmt.Errorf("integration %s configuration index #%d is missing rule", integName, configName)
					} else if config.Rule != "" && integ.Rule != "" {
						ruleText = fmt.Sprintf("(%s) && (%s)", integ.Rule, config.Rule)
					} else {
						ruleText = config.Rule
					}

					// Template merging. Choose the first from configuration,
					// integration, builtin.
					if template = utils.FirstNonEmpty(config.Template, integ.Template, builtinInteg.template); template == "" {
						return nil, fmt.Errorf("integration %s must have a configured template", integName)
					}

					// Vars merging configuration overrides integration
					// overrides builtin.
					vars := utils.MergeMaps(builtinInteg.vars, integ.Vars, config.Vars)
					resolveVars(vars)

					rule, err := govaluate.NewEvaluableExpressionWithFunctions(ruleText, functions)
					if err != nil {
						return nil, fmt.Errorf("error constructing rule for %s: %s", integName, err)
					}

					// Labels merging. Merge the configuration and integration
					// labels if present. If neither specify labels then default
					// to the builtin.
					if integ.Labels == nil && config.Labels == nil {
						labels = builtinInteg.labels
					} else {
						if integ.Labels != nil {
							labels = *integ.Labels
						}

						if config.Labels != nil {
							labels = append(labels, *config.Labels...)
						}
					}

					// If an integration specifies a specific plugin name use that.
					// Otherwise use the integration name/type. This is used for JMX
					// integrations that are separate integrations but all get grouped
					// by a jmx plugin type.
					plugin := integ.Plugin
					if plugin == "" {
						plugin = string(integName)
					}

					configs = append(configs, &configuration{services.ServiceType(integName), ruleText, rule,
						vars, template, plugin, labels})
				}
			}
		}
	}

	// Now load builtin configurations if the integration wasn't previously configured.
	log.Printf("user-configured integrations: %v", reflect.ValueOf(configured).MapKeys())

	disabled := map[services.ServiceType]bool{}
	for _, config := range overrides {
		for _, integ := range config.Disabled {
			disabled[services.ServiceType(integ)] = true
		}
	}

	log.Printf("disabled integration: %v", reflect.ValueOf(disabled).MapKeys())

	for integName, builtin := range builtinsMap {
		if !configured[integName] && !disabled[integName] && builtin.rule != nil {
			configured[integName] = true
			configs = append(configs, builtin)
		}
	}

	log.Printf("loaded %d configurations for %d integrations: %v", len(configs), len(configured),
		reflect.ValueOf(configured).MapKeys())

	return configs, nil
}

// Configure the filter
func (f *Filter) Configure(cfg *viper.Viper) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	ws := config.NewAssetSpec()
	for _, dir := range []string{"builtins", "overrides"} {
		path := cfg.GetString(dir)
		if path != "" {
			ws.Dirs[dir] = path
		}
	}
	log.Printf("%+v", ws)

	if err := f.assets.Update(ws); err != nil {
		log.Printf("error updating watches: %s", err)
	}

	return nil
}

func (f *Filter) onAssetChange(assets *config.AssetsView) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if err := f.load(assets); err != nil {
		log.Printf("failed loading integrations config: %s", err)
	}
}

func (f *Filter) load(assets *config.AssetsView) error {
	var builtinConfigs, overrideConfigs []configFile

	if builtins, ok := assets.Dirs["builtins"]; ok {
		builtins, err := readConfigs(builtins)
		if err != nil {
			return err
		}

		builtinConfigs, err = loadConfigs(builtins)
		if err != nil {
			return err
		}
	} else {
		return errors.New("integration builtins are required")
	}

	// Ensure that builtins have loaded before loading overrides
	if len(builtinConfigs) <= 0 {
		log.Printf("waiting for built in configurations to load before loading overrides")
	} else if overrides, ok := assets.Dirs["overrides"]; ok {

		overrides, err := readConfigs(overrides)
		if err != nil {
			return err
		}
		overrideConfigs, err = loadConfigs(overrides)

		if err != nil {
			return err
		}
	} else {
		log.Printf("no integrations overrides were provided")
	}

	configs, err := buildConfigurations(builtinConfigs, overrideConfigs)
	if err != nil {
		return err
	}

	f.configurations = configs

	return nil
}

// Matches if service instance satisfies rules
func matches(si *services.Instance, expr *govaluate.EvaluableExpression) bool {
	sm := map[string]interface{}{
		"ContainerID":        si.Container.ID,
		"ContainerName":      si.Container.Names[0],
		"ContainerImage":     si.Container.Image,
		"ContainerPod":       si.Container.Pod,
		"ContainerCommand":   si.Container.Command,
		"ContainerState":     si.Container.State,
		"ContainerNamespace": si.Container.Namespace,
		"NetworkIP":          si.Port.IP,
		"NetworkType":        si.Port.Type,
		"NetworkPublicPort":  float64(si.Port.PublicPort),
		"NetworkPrivatePort": float64(si.Port.PrivatePort),
		"ContainerLabels":    si.Container.Labels,
		"NetworkLabels":      si.Port.Labels,
	}

	ret, err := expr.Evaluate(sm)
	if err != nil {
		log.Printf("error evaluating match: %s", err)
		return false
	}

	exprVal, ok := ret.(bool)
	if ok {
		return exprVal
	}

	log.Printf("error: expression did not return boolean")
	return false
}

// Shutdown filter
func (f *Filter) Shutdown() {
	f.assets.Stop()
}

// Map takes discovered service instances and applies integration-specific configurations
func (f *Filter) Map(sis services.Instances) (services.Instances, error) {
	var instances services.Instances

Outer:
	for _, si := range sis {
		for _, config := range f.configurations {
			if matches(&si, config.rule) {
				si.Vars = config.vars
				si.Template = config.template
				si.Service.Type = config.serviceType
				si.Service.Plugin = config.plugin
				for _, label := range config.labels {
					if val, ok := si.Container.Labels[label]; ok {
						si.Orchestration.Dims[label] = val
					}
				}
				instances = append(instances, si)
				continue Outer
			}
		}
	}

	return instances, nil
}

