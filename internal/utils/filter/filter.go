// Package filter contains common filtering logic that can be used to filter
// datapoints or various resources within other agent components, such as
// monitors.  Filter instances have a Matches function which takes an instance
// of the type that they filter and return whether that instance matches the
// filter.
package filter

import (
    "regexp"
    "strings"
)

// StringFilter matches against simple strings
type StringFilter interface {
	Matches(string) bool
}

// StringMapFilter matches against the values of a map[string]string.
type StringMapFilter interface {
	Matches(map[string]string) bool
}

type basicStringFilter struct {
	staticSet map[string]bool
	regexps   []*regexp.Regexp
}

type basicStringNegationFilter struct {
    negationFilter  StringFilter
}

// NewStringFilter returns a filter that can match against the provided items.
func NewStringFilter(items []string) (StringFilter, error) {
	staticSet := make(map[string]bool)
	var regexps []*regexp.Regexp
	for _, m := range items {
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
				return nil, err
			}

			regexps = append(regexps, re)
		} else {
			staticSet[m] = true
		}
	}

	return &basicStringFilter{
		staticSet: staticSet,
		regexps:   regexps,
	}, nil
}

func (f *basicStringFilter) Matches(s string) bool {
	return f.staticSet[s] || anyRegexMatches(s, f.regexps)
}

// NewStringMapFilter returns a filter that matches against the provided map.
// All key/value pairs must match the spec given in m for a map to be
// considered a match.
func NewStringMapFilter(m map[string]string) (StringMapFilter, error) {
	staticSet := map[string]string{}
	regexps := map[string]*regexp.Regexp{}
	for k, v := range m {
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
				return nil, err
			}

			regexps[k] = re
		} else {
			staticSet[k] = v
		}
	}

	return &fullStringMapFilter{
		staticSet: staticSet,
		regexps:   regexps,
	}, nil
}

// StripNegation checks if a string is prefixed with "!"
// and will returned the stripped string and true if so
// else, return original value and false
func stripNegation(value string) (string, bool) {
   if strings.HasPrefix(value, "!") {
       return value[1:], true
   }
   return value, false
}

// Each key/value pair must match the filter for the whole map to match.
type fullStringMapFilter struct {
	staticSet map[string]string
	regexps   map[string]*regexp.Regexp
}

func (f *fullStringMapFilter) Matches(m map[string]string) bool {
	for k, v := range f.staticSet {
		if m[k] != v {
			return false
		}
	}
	for k, re := range f.regexps {
		if _, present := m[k]; !present {
			return false
		}
		if !re.MatchString(m[k]) {
			return false
		}
	}
	return true
}
