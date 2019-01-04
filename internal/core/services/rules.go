package services

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"

	"github.com/Knetic/govaluate"
)

var errNoValueFound = errors.New("no value was found in the map with the key")

// get returns the value of the specified key in the supplied map
func get(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, errors.New("Get takes 2 args")
	}
	inputMap := args[0]
	key := args[1]

	var defVal interface{}
	if len(args) == 3 {
		defVal = args[2]
	}

	interfaceMap, ok := inputMap.(map[interface{}]interface{})
	if !ok {
		return nil, errors.New("label must be a map[string]string")
	}

	keyStr, ok := key.(string)
	if !ok {
		return nil, errors.New("label must be of type string")
	}

	stringMap := utils.InterfaceMapToStringMap(interfaceMap)
	if val, ok := stringMap[keyStr]; ok {
		return val, nil
	} else if defVal != nil {
		return defVal, nil
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

func evaluateRule(si Endpoint, ruleText string, errorOnMissing bool) (interface{}, error) {
	rule, err := parseRuleText(ruleText)
	if err != nil {
		return nil, errors.WithMessage(err, "Could not parse rule")
	}

	asMap := utils.DuplicateInterfaceMapKeysAsCamelCase(EndpointAsMap(si))
	if err := endpointMapHasAllVars(asMap, rule.Vars()); err != nil {
		// If there are missing vars
		if !errorOnMissing {
			return nil, nil
		}
	}

	return rule.Evaluate(asMap)
}

// DoesServiceMatchRule returns true if service endpoint satisfies the rule
// given
func DoesServiceMatchRule(si Endpoint, ruleText string) bool {
	ret, err := evaluateRule(si, ruleText, false)
	if err != nil {
		log.WithFields(log.Fields{
			"discoveryRule":   ruleText,
			"serviceInstance": spew.Sdump(si),
			"error":           err,
		}).Error("Could not evaluate discovery rule")
		return false
	}

	if ret == nil {
		return false
	}
	exprVal, ok := ret.(bool)
	if !ok {
		log.WithFields(log.Fields{
			"discoveryRule":   ruleText,
			"serviceInstance": spew.Sdump(si),
		}).Errorf("Discovery rule did not evaluate to a true/false value")
		return false
	}

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
