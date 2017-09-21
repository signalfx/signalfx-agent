package utils

import (
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
