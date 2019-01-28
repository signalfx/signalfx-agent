package filter

// ExhaustiveStringFilter matches input strings that are positively matched by
// one of the input filters AND are not excluded by any negated filters (they
// work kind of like how .gitignore patterns work), OR are exactly matched by a
// literal filter input (e.g. not a globbed or regex pattern).  Order of the
// items does not matter.
type ExhaustiveStringFilter struct {
	*BasicStringFilter
}

// NewExhaustiveStringFilter makes a new ExhaustiveStringFilter with the given
// items.
func NewExhaustiveStringFilter(items []string) (*ExhaustiveStringFilter, error) {
	basic, err := NewBasicStringFilter(items)
	if err != nil {
		return nil, err
	}

	return &ExhaustiveStringFilter{
		BasicStringFilter: basic,
	}, nil
}

// Matches if s is positively matched by the filter items AND is not excluded
// by any, OR if it is postively matched by a non-glob/regex pattern exactly
// and is negated as well.  See the unit tests for examples.
func (f *ExhaustiveStringFilter) Matches(s string) bool {
	staticPositiveMatch := false
	staticNegativeMatch := false
	for val, negated := range f.staticSet {
		hit := val == s
		if negated {
			staticNegativeMatch = staticNegativeMatch || hit
		} else {
			staticPositiveMatch = staticPositiveMatch || hit
		}
	}

	regexPositiveMatch := false
	regexNegativeMatch := false
	for _, reMatch := range f.regexps {
		hit := reMatch.re.MatchString(s)
		if reMatch.negated {
			regexNegativeMatch = regexNegativeMatch || hit
		} else {
			regexPositiveMatch = regexPositiveMatch || hit
		}
	}
	return staticPositiveMatch || (regexPositiveMatch && !(staticNegativeMatch || regexNegativeMatch))
}
