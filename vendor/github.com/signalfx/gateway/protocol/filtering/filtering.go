package filtering

import (
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	"regexp"
	"sync/atomic"
)

// FilteredForwarder is a struct to hold the filtering logic
type FilteredForwarder struct {
	allow              []*regexp.Regexp
	deny               []*regexp.Regexp
	FilteredDatapoints int64
}

// FilterObj contains the Allow and Deny objects
type FilterObj struct {
	Allow []string `json:",omitempty"`
	Deny  []string `json:",omitempty"`
}

// Setup the FilteredForwarder based on the FilteredForwarderConfig
func (f *FilteredForwarder) Setup(filters *FilterObj) error {
	if filters != nil {
		allows := make([]*regexp.Regexp, len(filters.Allow))
		denys := make([]*regexp.Regexp, len(filters.Deny))
		for i, a := range filters.Allow {
			ra, err := regexp.Compile(a)
			if err != nil {
				return err
			}
			allows[i] = ra
		}
		for i, d := range filters.Deny {
			rd, err := regexp.Compile(d)
			if err != nil {
				return err
			}
			denys[i] = rd
		}
		f.allow = allows
		f.deny = denys
	}
	return nil
}

// FilterMetricName returns true for a metric which matches allow, or if no allow rules are present,
// if it didn't match deny. Returns false otherwise.
func (f *FilteredForwarder) FilterMetricName(metricName string) bool {
	denied := false
	for _, a := range f.deny {
		if a.Match([]byte(metricName)) {
			denied = true
			break
		}
	}
	found := false
	for _, a := range f.allow {
		if a.Match([]byte(metricName)) {
			denied = false
			found = true
			break
		}
	}
	return found || (len(f.allow) == 0 && !denied)
}

// FilterDatapoints filters datapoints based on the metric name as well as counts how many it filters
func (f *FilteredForwarder) FilterDatapoints(datapoints []*datapoint.Datapoint) []*datapoint.Datapoint {
	// TODO use a sync.pool of buffers here
	// TODO if we spun this off into several go routines instead of doing this in the main forwarder thread we could do a lot more work
	validDatapoints := make([]*datapoint.Datapoint, 0, len(datapoints))
	for _, d := range datapoints {
		if f.FilterMetricName(d.Metric) {
			validDatapoints = append(validDatapoints, d)
		}
	}
	invalidDatapoints := len(datapoints) - len(validDatapoints)
	atomic.AddInt64(&f.FilteredDatapoints, int64(invalidDatapoints))
	return validDatapoints
}

// GetFilteredDatapoints returns a cumulative counter of how many datapoints were filtered by this forwarder
func (f *FilteredForwarder) GetFilteredDatapoints() []*datapoint.Datapoint {
	return []*datapoint.Datapoint{sfxclient.Cumulative("filtered_by_forwarder", nil, atomic.LoadInt64(&f.FilteredDatapoints))}
}
