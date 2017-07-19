// A filter is what filters out unwanted metrics.  It is configured from the
// agent configuration file and is intended to be passed into each monitor for
// use if it sends datapoints on its own..
package filters

import (
	"regexp"

	"github.com/signalfx/golib/datapoint"
	log "github.com/sirupsen/logrus"
)

type Filter struct {
	monitorType string
	// These are all exclusion filters
	staticDimensionSet map[string]bool
	dimensionRegexps   map[string][]*regexp.Regexp
	staticMetricSet    map[string]bool
	metricRegexps      []*regexp.Regexp
}

func New(monitorType string, metricNames []string, dimensions map[string][]string) *Filter {
	staticDimensionSet := make(map[string]bool)
	dimensionRegexps := make(map[string][]*regexp.Regexp)

	for dimName, values := range dimensions {
		for _, v := range values {
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

				dimensionRegexps[dimName] = append(dimensionRegexps[dimName], re)
			} else {
				staticDimensionSet[dimKeyName(dimName, v)] = true
			}
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

	return &Filter{
		staticMetricSet:    staticMetricSet,
		dimensionRegexps:   dimensionRegexps,
		staticDimensionSet: staticDimensionSet,
		metricRegexps:      metricRegexps,
	}
}

func dimKeyName(dimName, value string) string {
	return dimName + ":" + value
}

func (f *Filter) Matches(dp *datapoint.Datapoint, monitorType string) bool {
	if f.monitorType != "" && monitorType != f.monitorType {
		return false
	}

	metricNamesMatch := !f.staticMetricSet[dp.Metric] && !anyRegexMatches(dp.Metric, f.metricRegexps)

	dimensionsMatch := false
	for dimName, value := range dp.Dimensions {
		staticNameExcluded := f.staticDimensionSet[dimKeyName(dimName, value)]
		regexpNameExcluded := anyRegexMatches(value, f.dimensionRegexps[dimName])
		if staticNameExcluded || regexpNameExcluded {
			dimensionsMatch = true
			break
		}
	}

	return metricNamesMatch && dimensionsMatch
}
