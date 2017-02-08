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
