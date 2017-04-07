package utils

// FirstNonEmpty returns the first string that is not empty, otherwise ""
func FirstNonEmpty(s ...string) string {
	for _, str := range s {
		if str != "" {
			return str
		}
	}

	return ""
}
