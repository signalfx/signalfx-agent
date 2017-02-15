package neotest

import (
	"encoding/json"
	"io/ioutil"
	"testing"
)

// LoadJSON unmarshals JSON for a test
func LoadJSON(t *testing.T, path string, dst interface{}) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s", path)
	}
	if err := json.Unmarshal(bytes, dst); err != nil {
		t.Fatalf("failed to unmarshal %s", path)
	}
}
