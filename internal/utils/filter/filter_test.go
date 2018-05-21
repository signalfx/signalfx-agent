package filter

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
)

func TestStringFilter(t *testing.T) {
	for _, tc := range []struct {
		filter      []string
		input       string
		shouldMatch bool
		shouldError bool
	}{
		{
			filter:      []string{},
			input:       "process_",
			shouldMatch: false,
		},
		{
			filter: []string{
				"/^process_/",
				"/^node_/",
			},
			input:       "process_",
			shouldMatch: true,
		},
		{
			filter: []string{
				"asdfdfasdf",
				"/^node_/",
			},
			input:       "process_",
			shouldMatch: false,
		},
	} {
		f, err := NewStringFilter(tc.filter)
		if tc.shouldError {
			assert.NotNil(t, err, spew.Sdump(tc))
		} else {
			assert.Nil(t, err, spew.Sdump(tc))
		}

		assert.Equal(t, tc.shouldMatch, f.Matches(tc.input), "%s\n%s", spew.Sdump(tc), spew.Sdump(f))
	}
}
