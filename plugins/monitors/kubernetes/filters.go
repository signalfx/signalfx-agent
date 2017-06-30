package kubernetes

type ExclusionSet map[string]bool

func NewExclusionSet(ss []string) ExclusionSet {
	set := make(map[string]bool, len(ss))
	for _, s := range ss {
		set[s] = true
	}
	return set
}

// An empty set passes everything
func (es ExclusionSet) IsExcluded(s string) bool {
	return len(es) != 0 && es[s]
}
