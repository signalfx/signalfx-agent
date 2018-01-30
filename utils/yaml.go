package utils

import (
	"reflect"
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

func YAMLNameOfField(field reflect.StructField) string {
	tmp := reflect.New(reflect.StructOf([]reflect.StructField{field})).Elem()
	asYaml, _ := yaml.Marshal(tmp.Interface())
	parts := strings.SplitN(string(asYaml), ":", 2)
	if parts[0] == string(asYaml) {
		return ""
	}
	return parts[0]
}
