package services

import (
	"errors"
	"fmt"

	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"

	"github.com/Knetic/govaluate"
)

var errNoValueFound = errors.New("no value was found in the map with the key")

// get returns the value of the specified key in the supplied map
func get(args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.New("Get takes 2 args")
	}
	inputMap := args[0]
	key := args[1]

	labelInterfaceMap, ok := inputMap.(map[interface{}]interface{})
	if !ok {
		return nil, errors.New("label must be a map[string]string")
	}

	label, ok := key.(string)
	if !ok {
		return nil, errors.New("label must be of type string")
	}

	labelMap := utils.InterfaceMapToStringMap(labelInterfaceMap)
	if val, ok := labelMap[label]; ok {
		return val, nil
	}

	return nil, nil
}

var ruleFunctions = map[string]govaluate.ExpressionFunction{
	"Get": get,
	"Contains": func(args ...interface{}) (interface{}, error) {
		val, err := get(args...)
		if err != nil {
			return false, err
		}
		return val != nil, nil
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

	asMap := utils.DuplicateInterfaceMapKeysAsCamelCase(EndpointAsMap(si))
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
