package kubernetes

import "strings"

// Describes both positive and negative filtration of string values.  
type FilterSet struct {
	inclusions map[string]bool
	exclusions map[string]bool
}

// If an element of ss starts with "!" it is added to the exclusion set,
// otherwise it is included.  If there are no inclusion filters, then
// everything not in the exclusion filter will pass.  If there are inclusion
// filters, then a string must be in the inclusion filter and not in the
// exclusion filter to pass.
func NewFilterSet(ss []string) *FilterSet {
	exclusions := make(map[string]bool)
	inclusions := make(map[string]bool)
	for _, s := range ss {
		if strings.HasPrefix(s, "!") {
			exclusions[strings.TrimLeft(s, "!")] = true
		} else {
			inclusions[s] = true
		}
	}
	return &FilterSet{
		inclusions: inclusions,
		exclusions: exclusions,
	}
}

// Returns whether the string should be filtered out
func (fs FilterSet) IsExcluded(s string) bool {
	if len(fs.inclusions) > 0 {
		return !fs.inclusions[s] || fs.exclusions[s]
	} else {
		return fs.exclusions[s]
	}
}
