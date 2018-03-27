package dpfilters

import "github.com/signalfx/golib/datapoint"

// FilterSet is a collection of datapont filters, any one of which must match
// for a datapoint to be matched.
type FilterSet struct {
	Filters []DatapointFilter
}

// Matches sends a datapoint through each of the filters in the set and returns
// true if at least one of them matches the datapoint.
func (fs *FilterSet) Matches(dp *datapoint.Datapoint) bool {
	for _, f := range fs.Filters {
		if f.Matches(dp) {
			return true
		}
	}
	return false
}
