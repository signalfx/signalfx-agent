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

func TestMapFunctions(t *testing.T) {
	interfacemap := map[interface{}]interface{}{"hello": "world", "good": "bye"}
	stringmap := map[string]interface{}{"hello": "world", "good": "bye"}
	t.Run("Get()", func(t *testing.T) {
		var val interface{}
		var err error
		_, err = ruleFunctions["Get"](stringmap, "hello")
		assert.Error(t, err, "should error out when wrong map type is used")

		_, err = ruleFunctions["Get"](interfacemap, 3)
		assert.Error(t, err, "should error out when wrong key type is used")

		_, err = ruleFunctions["Get"](interfacemap)
		assert.Error(t, err, "should error out when not enough arguments are provided")

		val, err = ruleFunctions["Get"](interfacemap, "nokey")
		assert.Error(t, err, "should error out if the map does not contain the desired value")
		assert.NotEqual(t, "world", val, "should return the expected value")

		val, err = ruleFunctions["Get"](interfacemap, "hello")
		assert.NoError(t, err, "should not error out if the map contains the desired value")
		assert.Equal(t, "world", val, "should return the expected value")
	})
	t.Run("Contains()", func(t *testing.T) {
		val, err := ruleFunctions["Contains"](interfacemap, "nokey")
		assert.NoError(t, err, "should not error if the supplied arguments are the correct type")
		assert.False(t, val.(bool), "should return false when an error occurs")

		val, err = ruleFunctions["Contains"](stringmap)
		assert.Error(t, err, "should error if the supplied arguments are the wrong type")
		assert.False(t, val.(bool), "should return false when an error occurs")

		val, err = ruleFunctions["Contains"](interfacemap, "good")
		assert.NoError(t, err, "should not error out if the map contains the desired value")
		assert.True(t, val.(bool), "should return the expected value")
	})
}
