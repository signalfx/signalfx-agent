package utils

import (
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
