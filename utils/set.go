package utils

// UniqueStrings returns a slice with the unique set of strings from the input
func UniqueStrings(strings []string) []string {
	unique := map[string]struct{}{}
	for _, v := range strings {
		unique[v] = struct{}{}
	}

	keys := make([]string, 0)
	for k := range unique {
		keys = append(keys, k)
	}

	return keys
}

// StringSliceToMap converts a slice of strings into a map with keys from the slice
func StringSliceToMap(strings []string) map[string]bool {
	// Use bool so that the user can do `if setMap[key] { ... }``
	ret := map[string]bool{}
	for _, s := range strings {
		ret[s] = true
	}
	return ret
}
