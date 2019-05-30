package services

import (
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"

	"github.com/Knetic/govaluate"
)

// get returns the value of the specified key in the supplied map
func get(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, errors.New("Get takes at least 2 args")
	}
	inputMap := args[0]
	key := args[1]

	var defVal interface{}
	if len(args) == 3 {
		defVal = args[2]
	}

	mapVal := reflect.ValueOf(inputMap)
	if mapVal.Kind() != reflect.Map {
		return nil, errors.New("first arg to Get must be a map")
	}

	keyVal := reflect.ValueOf(key)

	if val := mapVal.MapIndex(keyVal); val.IsValid() && val.CanInterface() {
		return val.Interface(), nil
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
	"ToString": func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, errors.New("ToString takes exactly one parameter")
		}
		return fmt.Sprintf("%v", args[0]), nil
	},
}

func parseRuleText(text string) (*govaluate.EvaluableExpression, error) {
	return govaluate.NewEvaluableExpressionWithFunctions(text, ruleFunctions)
}

// EvaluateRule executes a govaluate expression against an endpoint
func EvaluateRule(si Endpoint, ruleText string, errorOnMissing bool, doValidation bool) (interface{}, error) {
	rule, err := parseRuleText(ruleText)
	if err != nil {
		return nil, errors.WithMessage(err, "Could not parse rule")
	}

	asMap := utils.DuplicateInterfaceMapKeysAsCamelCase(EndpointAsMap(si))
	if err := endpointMapHasAllVars(asMap, rule.Vars()); err != nil {
		// If there are missing vars
		if !errorOnMissing {
			if doValidation {
				log.WithField("discoveryRule", ruleText).Warnf(err.Error())
			}
			return nil, nil
		}
	}

	log.WithFields(log.Fields{
		"ruleText": ruleText,
		"asMap":    spew.Sdump(asMap),
	}).Debug("Evaluating rule")
	return rule.Evaluate(asMap)
}

// DoesServiceMatchRule returns true if service endpoint satisfies the rule
// given
func DoesServiceMatchRule(si Endpoint, ruleText string, doValidation bool) bool {
	ret, err := EvaluateRule(si, ruleText, false, doValidation)
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
		return fmt.Errorf("syntax error in discovery rule '%s': %s", rule, err.Error())
	}
	return nil
}

func endpointMapHasAllVars(endpointParams map[string]interface{}, vars []string) error {
	for _, v := range vars {
		if _, ok := endpointParams[v]; !ok {
			return fmt.Errorf("variable '%s' not found in endpoint", v)
		}
	}
	return nil
}
