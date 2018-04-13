package metrics

import "strings"

var propNameSanitizer = strings.NewReplacer(
	".", "_",
	"/", "_")

func propsAndTagsFromLabels(labels map[string]string) (map[string]string, map[string]bool) {
	props := make(map[string]string)
	tags := make(map[string]bool)

	for label, value := range labels {
		key := propNameSanitizer.Replace(label)
		// K8s labels without values are treated as tags
		if value == "" {
			tags[key] = true
		} else {
			props[key] = value
		}
	}

	return props, tags
}
