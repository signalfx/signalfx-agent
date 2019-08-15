package propfilters

import (
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
)

// FilterSet is a collection of property filters, any one of which must match
// for a property to be matched.
type FilterSet struct {
	Filters []DimPropsFilter
}

// FilterDimProps sends a *types.Dimension through each of the filters in the set
// and filters properties. All original properties will be returned if no filter matches
//, or a subset of the original if some are filtered, or nil if all are filtered.
func (fs *FilterSet) FilterDimProps(dimProps *types.Dimension) *types.Dimension {
	filteredDimProps := &(*dimProps)
	for _, f := range fs.Filters {
		filteredDimProps = f.FilterDimProps(filteredDimProps)
	}
	return filteredDimProps
}
