package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDoesServiceMatchRule(t *testing.T) {
	t.Run("Handles parse error in discovery rule", func(t *testing.T) {
		endpoint := NewEndpointCore("abcd", "test", "test")
		assert.False(t, DoesServiceMatchRule(endpoint, "== ++ abc 1jj +"))
	})
}
