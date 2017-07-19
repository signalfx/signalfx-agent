package filters

import "github.com/signalfx/golib/datapoint"

type FilterSet struct {
	Filters []*Filter
}

func (fs *FilterSet) Matches(dp *datapoint.Datapoint, monitorType string) bool {
	for _, f := range fs.Filters {
		if !f.Matches(dp, monitorType) {
			return false
		}
	}
	return true
}
