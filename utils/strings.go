package utils

import (
	"regexp"
	"strings"
	"unicode"
)

// FirstNonEmpty returns the first string that is not empty, otherwise ""
func FirstNonEmpty(s ...string) string {
	for _, str := range s {
		if str != "" {
			return str
		}
	}

	return ""
}

// FirstNonZero returns the first int in `ns` that is not zero.
func FirstNonZero(ns ...int) int {
	for _, n := range ns {
		if n != 0 {
			return n
		}
	}
	return 0
}

// IndentLines indents all lines in `ss` by `spaces` number of spaces
func IndentLines(ss string, spaces int) string {
	var output string
	for i := range ss {
		if i == 0 {
			output += strings.Repeat(" ", spaces) + string(ss[i])
		} else if ss[i] == '\n' && i != len(ss)-1 {
			output += "\n" + strings.Repeat(" ", spaces)
		} else {
			output += string(ss[i])
		}
	}
	return output
}

// LowercaseFirstChar make the first character of a string lowercase
func LowercaseFirstChar(s string) string {
	for i, v := range s {
		return string(unicode.ToLower(v)) + s[i+1:]
	}
	return ""
}

// StripIndent looks at the first line in s and strips off whatever whitespace
// indentation it has from every line in s.  If subsequent lines do not start
// with the same indentation as the first line, results are undefined.
// If the first line is blank, it will be removed before processing.
func StripIndent(s string) string {
	lines := strings.Split(strings.TrimLeft(s, "\n"), "\n")
	re := regexp.MustCompile(`^(\s+)`)
	matches := re.FindStringSubmatch(lines[0])
	if len(matches) > 0 {
		indent := matches[1]
		for i := range lines {
			lines[i] = strings.Replace(lines[i], indent, "", 1)
		}
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}
