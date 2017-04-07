package utils

// MergeMaps merges n maps with a later map's keys overriding earlier maps
func MergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	ret := map[string]interface{}{}

	for _, m := range maps {
		for k, v := range m {
			ret[k] = v
		}
	}

	return ret
}
