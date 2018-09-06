package utils

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

// ConvertToMapViaYAML takes a struct and converts it to map[string]interface{}
// by marshalling it to yaml and back to a map.  This will return nil if the
// conversion was not successful.
func ConvertToMapViaYAML(obj interface{}) (map[string]interface{}, error) {
	str, err := yaml.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var newMap map[string]interface{}
	if err := yaml.Unmarshal(str, &newMap); err != nil {
		return nil, err
	}

	return newMap, nil
}

// YAMLNameOfField returns the YAML key that is used for the given struct
// field.  It does this by actually serializing the field and parsing the
// output string.  If the field has no key (e.g. if the `yaml:"-"` tag is set,
// this will return an empty string.
func YAMLNameOfField(field reflect.StructField) string {
	if strings.HasPrefix(field.Tag.Get("yaml"), ",inline") {
		return ""
	}
	tmp := reflect.New(reflect.StructOf([]reflect.StructField{field})).Elem()
	asYaml, _ := yaml.Marshal(tmp.Interface())
	parts := strings.SplitN(string(asYaml), ":", 2)
	if parts[0] == string(asYaml) {
		return ""
	}
	return parts[0]
}

// YAMLNameOfFieldInStruct returns the YAML key that is used for the given
// struct field, looking up fieldName in the given st struct.  If the field has
// no key (e.g. if the `yaml:"-"` tag is set, this will return an empty string.
// It uses YAMLNameOfField under the covers.  If st is not a struct, this will
// panic.
func YAMLNameOfFieldInStruct(fieldName string, st interface{}) string {
	stType := reflect.Indirect(reflect.ValueOf(st)).Type()
	field, ok := stType.FieldByName(fieldName)
	if !ok {
		return ""
	}
	return YAMLNameOfField(field)
}

// ParseLineNumberFromYAMLError takes an error message nested in yaml.TypeError
// and returns a line number if indicated in the error message.  This is pretty
// hacky but is the only way to actually get at the line number in the standard
// yaml package.
func ParseLineNumberFromYAMLError(e string) int {
	re := regexp.MustCompile(`line (\d+): `)
	match := re.FindStringSubmatch(e)
	if len(match) > 0 {
		asInt, err := strconv.Atoi(match[1])
		if err != nil {
			return 0
		}
		return asInt
	}
	return 0
}

// DecodeValueGenerically apply some very basic heuristics to decode string values to the most
// sensible type for use in config structs.
func DecodeValueGenerically(val string) interface{} {
	// The literal values of true/false get interpreted as bools
	if val == "true" {
		return true
	}
	if val == "false" {
		return false
	}

	// Try to decode as an integer
	if asInt, err := strconv.Atoi(val); err == nil {
		return asInt
	}

	// See if it's an array/list
	if strings.HasPrefix(val, "[") {
		var out []interface{}
		if err := yaml.Unmarshal([]byte(val), &out); err == nil {
			return out
		}
	}

	// Next try to see if it's some kind of object and return the generic
	// yaml MapSlice so that it will be reserialized back to the original form
	// when injected to a monitor instance.  That way we don't have to have
	// knowledge about monitor config types here.
	if strings.HasPrefix(val, "{") {
		var out yaml.MapSlice
		if err := yaml.Unmarshal([]byte(val), &out); err == nil {
			return out
		}
	}

	// Otherwise just treat it as the string it always was
	return val
}
