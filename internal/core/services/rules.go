package services

import (
	"errors"
	"fmt"

	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"

	"github.com/Knetic/govaluate"
	"github.com/iancoleman/strcase"
)

var ruleFunctions = map[string]govaluate.ExpressionFunction{
	"Get": func(args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, errors.New("Get takes 2 args")
		}
		arg1 := args[0]
		arg2 := args[1]

		labelInterfaceMap, ok := arg1.(map[interface{}]interface{})
		if !ok {
			return nil, errors.New("label must be a map[string]string")
		}
		label, ok := arg2.(string)
		if !ok {
			return nil, errors.New("label must be of type string")
		}

		labelMap := utils.InterfaceMapToStringMap(labelInterfaceMap)
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

		labelInterfaceMap, ok := arg1.(map[interface{}]interface{})
		if !ok {
			return nil, errors.New("label must be a map[string]string")
		}

		label, ok := arg2.(string)
		if !ok {
			return nil, errors.New("label must be of type string")
		}

		labelMap := utils.InterfaceMapToStringMap(labelInterfaceMap)
		if _, ok := labelMap[label]; ok {
			return true, nil
		}

		return false, nil
	},
}

func parseRuleText(text string) (*govaluate.EvaluableExpression, error) {
	return govaluate.NewEvaluableExpressionWithFunctions(text, ruleFunctions)
}

// DoesServiceMatchRule returns true if service endpoint satisfies the rule
// given
func DoesServiceMatchRule(si Endpoint, ruleText string) bool {
	rule, err := parseRuleText(ruleText)
	if err != nil {
		log.WithFields(log.Fields{
			"discoveryRule": ruleText,
		}).Error("Could not parse discovery rule")
		return false
	}

	asMap := duplicateKeysAsCamelCase(EndpointAsMap(si))
	if err := endpointMapHasAllVars(asMap, rule.Vars()); err != nil {
		log.WithFields(log.Fields{
			"discoveryRule":   rule.String(),
			"values":          asMap,
			"serviceInstance": si,
			"error":           err,
		}).Debug("Endpoint does not include some variables used in rule, assuming does not match")
		return false
	}

	ret, err := rule.Evaluate(asMap)
	if err != nil {
		log.WithFields(log.Fields{
			"discoveryRule":   rule.String(),
			"values":          asMap,
			"serviceInstance": si,
			"error":           err,
		}).Error("Could not evaluate discovery rule")
		return false
	}

	exprVal, ok := ret.(bool)
	if !ok {
		log.WithFields(log.Fields{
			"discoveryRule": rule.String(),
			"values":        asMap,
		}).Errorf("Discovery rule did not evaluate to a true/false value")
		return false
	}

	log.WithFields(log.Fields{
		"rule":      ruleText,
		"variables": asMap,
		"result":    exprVal,
	}).Debug("Rule evaluated")

	return exprVal
}

func duplicateKeysAsCamelCase(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range m {
		out[k] = v
		out[strcase.ToLowerCamel(k)] = v
	}
	return out
}

// ValidateDiscoveryRule takes a discovery rule string and returns false if it
// can be determined to be invalid.  It does not guarantee validity but can be
// used to give upfront feedback to the user if there are syntax errors in the
// rule.
func ValidateDiscoveryRule(rule string) error {
	if _, err := parseRuleText(rule); err != nil {
		return fmt.Errorf("Syntax error in discovery rule '%s': %s", rule, err.Error())
	}
	return nil
}

func endpointMapHasAllVars(endpointParams map[string]interface{}, vars []string) error {
	for _, v := range vars {
		if _, ok := endpointParams[v]; !ok {
			return fmt.Errorf("Variable '%s' not found in endpoint", v)
		}
	}
	return nil
}
