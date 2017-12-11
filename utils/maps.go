// Package utils hold miscelaneous utility functions
package utils

import (
	"fmt"
	"sort"
)

// MergeStringMaps merges n maps with a later map's keys overriding earlier maps
func MergeStringMaps(maps ...map[string]string) map[string]string {
	ret := map[string]string{}

	for _, m := range maps {
		for k, v := range m {
			ret[k] = v
		}
	}

	return ret
}

// RemoveEmptyMapValues will strip a map of any key/value pairs for which the
// value is the empty string.
func RemoveEmptyMapValues(m map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range m {
		if v != "" {
			out[k] = v
		}
	}
	return out
}

// MergeInterfaceMaps merges any number of map[string]interface{} with a later
// map's keys overriding earlier maps.  Nil values do not override earlier
// values.
func MergeInterfaceMaps(maps ...map[string]interface{}) map[string]interface{} {
	ret := map[string]interface{}{}

	for i := range maps {
		for k, v := range maps[i] {
			if ret[k] == nil || v != nil {
				ret[k] = v
			}
		}
	}

	return ret
}

// CloneStringMap makes a shallow copy of a map[string]string
func CloneStringMap(m map[string]string) map[string]string {
	m2 := make(map[string]string)
	for k, v := range m {
		m2[k] = v
	}
	return m2
}

// InterfaceMapToStringMap converts a map[interface{}]interface{} to a
// map[string]string.  Keys and values will be converted with fmt.Sprintf so
// the original key/values don't have to be strings.
func InterfaceMapToStringMap(m map[interface{}]interface{}) map[string]string {
	out := make(map[string]string)
	for k, v := range m {
		out[fmt.Sprintf("%v", k)] = fmt.Sprintf("%v", v)
	}
	return out
}

// SortMapKeys returns a slice of all of the keys of a map sorted
// alphabetically ascending.
func SortMapKeys(m map[string]interface{}) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
