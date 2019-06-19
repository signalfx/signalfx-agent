package dpfilters

import "github.com/signalfx/golib/datapoint"

// AlwaysMatchFilter is a trivial filter that always matches datapoints
type AlwaysMatchFilter struct{}

// Matches just always returns true
func (m *AlwaysMatchFilter) Matches(*datapoint.Datapoint) bool {
	return true
}
