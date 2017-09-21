package filters

import (
	"regexp"
	"strings"
)

// Contains all of the logic for glob and regex based filtering

func isGlobbed(s string) bool {
	return strings.ContainsAny(s, "*?")
}

func isRegex(s string) bool {
	return len(s) > 2 && s[0] == '/' && s[len(s)-1] == '/'
}

// remove the bracketing slashes for a regex
func stripSlashes(s string) string {
	if len(s) < 2 {
		return s
	}

	return s[1 : len(s)-1]
}

func convertGlobToRegexp(g string) (*regexp.Regexp, error) {
	reText := ""
	for _, ch := range g {
		if ch == '*' {
			reText += ".*"
		} else if ch == '?' {
			reText += ".?"
		} else if ch == '(' {
			reText += "\\("
		} else if ch == ')' {
			reText += "\\)"
		} else if ch == '.' {
			reText += "\\."
		} else {
			reText += string(ch)
		}
	}

	return regexp.Compile(reText)
}

func anyRegexMatches(s string, res []*regexp.Regexp) bool {
	for _, re := range res {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}
