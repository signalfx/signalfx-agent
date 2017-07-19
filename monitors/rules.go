package monitors

import (
	"errors"

	"github.com/signalfx/neo-agent/observers"
	log "github.com/sirupsen/logrus"

	"github.com/Knetic/govaluate"
)

// Used to quickly check if a discovery rule is invalid without actually
// evaluating it
var validRuleIdentifiers = map[string]bool{
	"ContainerID":        true,
	"ContainerName":      true,
	"ContainerImage":     true,
	"ContainerCommand":   true,
	"ContainerState":     true,
	"IP":                 true,
	"NetworkType":        true,
	"NetworkPublicPort":  true,
	"NetworkPrivatePort": true,
	"ContainerLabels":    true,
	"NetworkLabels":      true,
	"Pod":                true,
	"Namespace":          true,
}

var ruleFunctions = map[string]govaluate.ExpressionFunction{
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

func parseRuleText(text string) (*govaluate.EvaluableExpression, error) {
	return govaluate.NewEvaluableExpressionWithFunctions(text, ruleFunctions)
}

// Matches if service instance satisfies rules
func doesServiceMatchRule(si *observers.ServiceInstance, ruleText string) bool {
	// TODO: consider caching parsed rule for maximum efficiency
	rule, err := parseRuleText(ruleText)
	if err != nil {
		log.WithFields(log.Fields{
			"discoveryRule": ruleText,
		}).Error("Could not parse discovery rule")
	}

	sm := map[string]interface{}{
		"ContainerID":        si.Container.ID,
		"ContainerName":      si.Container.PrimaryName(),
		"ContainerImage":     si.Container.Image,
		"ContainerCommand":   si.Container.Command,
		"ContainerState":     si.Container.State,
		"IP":                 si.Port.IP,
		"NetworkType":        si.Port.Type,
		"NetworkPublicPort":  float64(si.Port.PublicPort),
		"NetworkPrivatePort": float64(si.Port.PrivatePort),
		"ContainerLabels":    si.Container.Labels,
		"NetworkLabels":      si.Port.Labels,
		// K8s specific
		"Pod":       si.Container.Pod,
		"Namespace": si.Container.Namespace,
	}

	ret, err := rule.Evaluate(sm)
	if err != nil {
		log.WithFields(log.Fields{
			"discoveryRule":   rule.String(),
			"values":          sm,
			"serviceInstance": si,
		}).Error("Could not evaluate discovery rule")
		return false
	}

	exprVal, ok := ret.(bool)
	if !ok {
		log.WithFields(log.Fields{
			"discoveryRule": rule.String(),
			"values":        sm,
		}).Errorf("Discovery rule did not evaluate to a true/false value")
		return false
	}

	log.WithFields(log.Fields{
		"rule":      ruleText,
		"variables": sm,
		"result":    exprVal,
	}).Debug("Rule evaluated")

	return exprVal
}

// Map takes discovered service instances and applies integration-specific configurations
// TODO: figure out what that label/dim logic does
/*func (f *Filter) Map(sis services.Instances) (services.Instances, error) {
	var instances services.Instances

	for _, si := range sis {
		if doesServiceMatchRule(&si, config.rule) {
			for _, label := range config.labels {
				if val, ok := si.Container.Labels[label]; ok {
					si.Orchestration.Dims[label] = val
				}
			}
			instances = append(instances, si)
			continue
		}
	}

	return instances, nil
}*/
