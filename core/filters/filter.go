// Package filters has logic describing the filtering of unwanted metrics.  Filters
// are configured from the agent configuration file and is intended to be passed
// into each monitor for use if it sends datapoints on its own.
package filters

import (
	"regexp"

	"github.com/signalfx/golib/datapoint"
	log "github.com/sirupsen/logrus"
)

// Filter describes any datapoint filter
type Filter interface {
	// Matches takes a datapoint and returns whether it is matched by the
	// filter
	Matches(*datapoint.Datapoint) bool
}

// BasicFilter is an exclusionary filter that is designed to filter SignalFx
// datapoint objects.  It can filter based on the monitor type, dimensions, or
// the metric name.  It supports both static, globbed, and regex patterns for
// filter values.
// If dimensions are specifed, they must all match for the datapoint to match.
// If multiple metric names are given, only one must match for the datapoint to
// match the filter since datapoints can only have one metric name.
type BasicFilter struct {
	monitorType string
	// These are all exclusion filters
	staticDimensionSet map[string]string
	dimensionRegexps   map[string]*regexp.Regexp
	staticMetricSet    map[string]bool
	metricRegexps      []*regexp.Regexp
}

// New returns a new filter with the given configuration
func New(monitorType string, metricNames []string, dimensions map[string]string) *BasicFilter {
	staticDimensionSet := make(map[string]string)
	dimensionRegexps := make(map[string]*regexp.Regexp)

	for dimName, v := range dimensions {
		if isRegex(v) || isGlobbed(v) {
			var re *regexp.Regexp
			var err error

			if isRegex(v) {
				reText := stripSlashes(v)
				re, err = regexp.Compile(reText)
			} else {
				re, err = convertGlobToRegexp(v)
			}

			if err != nil {
				log.WithFields(log.Fields{
					"filter":     v,
					"filterType": "dimension",
					"error":      err,
				}).Error("Could not parse glob/regexp for filter")
				continue
			}

			dimensionRegexps[dimName] = re
		} else {
			staticDimensionSet[dimName] = v
		}
	}

	staticMetricSet := make(map[string]bool)
	var metricRegexps []*regexp.Regexp
	for _, m := range metricNames {
		if isRegex(m) || isGlobbed(m) {
			var re *regexp.Regexp
			var err error

			if isRegex(m) {
				reText := stripSlashes(m)
				re, err = regexp.Compile(reText)
			} else {
				re, err = convertGlobToRegexp(m)
			}

			if err != nil {
				log.WithFields(log.Fields{
					"filter":     m,
					"filterType": "metric",
					"error":      err,
				}).Error("Could not parse regexp for filter")
				continue
			}

			metricRegexps = append(metricRegexps, re)
		} else {
			staticMetricSet[m] = true
		}
	}

	return &BasicFilter{
		staticMetricSet:    staticMetricSet,
		metricRegexps:      metricRegexps,
		staticDimensionSet: staticDimensionSet,
		dimensionRegexps:   dimensionRegexps,
	}
}

func (f *BasicFilter) dimensionsMatch(dims map[string]string) bool {
	for dimName, value := range f.staticDimensionSet {
		if dims[dimName] != value {
			return false
		}
	}
	for dimName, re := range f.dimensionRegexps {
		if _, present := dims[dimName]; !present {
			return false
		}
		if !re.MatchString(dims[dimName]) {
			return false
		}
	}
	return true
}

func (f *BasicFilter) metricNameMatches(name string) bool {
	return (len(f.staticMetricSet) == 0 && len(f.metricRegexps) == 0) ||
		(f.staticMetricSet[name] || anyRegexMatches(name, f.metricRegexps))
}

// Matches tests a datapoint to see whether it is excluded by this filter.  In
// order to match on monitor type, the datapoint should have the "monitorType"
// key set in it's Meta field.
func (f *BasicFilter) Matches(dp *datapoint.Datapoint) bool {
	if f.monitorType != "" && dp.Meta["monitorType"] != f.monitorType {
		return false
	}

	return f.metricNameMatches(dp.Metric) && f.dimensionsMatch(dp.Dimensions)
}
