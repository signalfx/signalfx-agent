package dpfilters

import (
	"github.com/signalfx/golib/datapoint"
)

// FilterSet is a collection of datapont filters, any one of which must match
// for a datapoint to be matched.
type FilterSet struct {
	ExcludeFilters []DatapointFilter
	IncludeFilters []DatapointFilter
}

// Matches sends a datapoint through each of the filters in the set and returns
// true if at least one of them matches the datapoint.
func (fs *FilterSet) Matches(dp *datapoint.Datapoint) bool {
	for _, ex := range fs.ExcludeFilters {
		if ex.Matches(dp) {
			// If we match an exclusionary filter, run through each inclusion
			// filter and see if anything includes the metrics.
			for _, incl := range fs.IncludeFilters {
				if incl.Matches(dp) {
					return false
				}
			}
			return true
		}
	}
	return false
}
