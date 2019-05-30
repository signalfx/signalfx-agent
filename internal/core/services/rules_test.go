package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDoesServiceMatchRule(t *testing.T) {
	t.Run("Handles parse error in discovery rule", func(t *testing.T) {
		endpoint := NewEndpointCore("abcd", "test", "test", nil)
		assert.False(t, DoesServiceMatchRule(endpoint, "== ++ abc 1jj +", true))
	})
}

func TestMapFunctions(t *testing.T) {
	interfacemap := map[interface{}]interface{}{"hello": "world", "good": "bye"}
	stringmap := map[string]interface{}{"hello": "world", "good": "bye"}

	t.Run("Get() handles string -> interface{} map type", func(t *testing.T) {
		val, err := ruleFunctions["Get"](stringmap, "hello")
		assert.NoError(t, err, "should not error out if the map contains the desired value")
		assert.Equal(t, "world", val, "should return the expected value")
	})

	t.Run("Get() returns string -> interface{} map type", func(t *testing.T) {
		val, err := ruleFunctions["Get"](stringmap, "hello")
		assert.NoError(t, err, "should not error out if the map contains the desired value")
		assert.Equal(t, "world", val, "should return the expected value")
	})

	t.Run("Get() returns error on bad map type", func(t *testing.T) {
		_, err := ruleFunctions["Get"]("string", 3)
		assert.Error(t, err, "should error out when wrong map type is used")
	})

	t.Run("Get() insufficient number of arguments", func(t *testing.T) {
		_, err := ruleFunctions["Get"](interfacemap)
		assert.Error(t, err, "should error out when not enough arguments are provided")
	})

	t.Run("Get() map does not contain key", func(t *testing.T) {
		val, err := ruleFunctions["Get"](interfacemap, "nokey")
		assert.NoError(t, err, "should not error out if the map does not contain the desired value")
		assert.Nil(t, val, "should return nil if the map does not contain the desired value")
	})

	t.Run("Get() returns default if not in map", func(t *testing.T) {
		val, err := ruleFunctions["Get"](interfacemap, "nokey", 50)
		assert.NoError(t, err, "should not error out if the map does not contain the desired value")
		assert.Equal(t, val, 50, "should return default if the map does not contain the desired value")
	})

	t.Run("Get() handles interface{} -> interface{} maps", func(t *testing.T) {
		val, err := ruleFunctions["Get"](interfacemap, "hello")
		assert.NoError(t, err, "should not error out if the map contains the desired value")
		assert.Equal(t, "world", val, "should return the expected value")
	})

	t.Run("Contains() map does not contain key", func(t *testing.T) {
		val, err := ruleFunctions["Contains"](interfacemap, "nokey")
		assert.NoError(t, err, "should not error if the supplied arguments are the correct type")
		assert.False(t, val.(bool), "should only return false if an error occured")
	})

	t.Run("Contains() incorrect argument types", func(t *testing.T) {
		val, err := ruleFunctions["Contains"](stringmap)
		assert.Error(t, err, "should error if the supplied arguments are the wrong type")
		assert.False(t, val.(bool), "should return false when an error occurs")
	})

	t.Run("Contains() map contians desired value", func(t *testing.T) {
		val, err := ruleFunctions["Contains"](interfacemap, "good")
		assert.NoError(t, err, "should not error out if the map contains the desired value")
		assert.True(t, val.(bool), "should return the expected value")
	})
}
