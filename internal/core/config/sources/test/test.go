package test

import (
	"context"
	"sync"

	"github.com/signalfx/signalfx-agent/internal/core/config/types"
)

type testVal struct {
	value   map[string][]byte
	version uint64
	err     error
}

// TestConfigSource is a dummy implementation of a ConfigSource that can be
// used for testing.
type TestConfigSource struct {
	sync.Mutex
	valuesByPath map[string]testVal
	valueCh      chan testVal
}

var _ types.ConfigSource = new(TestConfigSource)

func NewTestConfigSource() *TestConfigSource {
	cs := &TestConfigSource{}
	go cs.Run()
	return cs
}

func (ts *TestConfigSource) Run() {
	for {
		select {
		case val := <-ts.valueCh:
		}
	}
}

func (ts *TestConfigSource) UpdateWatchedValue(path string, val map[string][]byte, version uint64) {
	ts.Lock()
	defer ts.Unlock()

	watcher := ts.watchers[path]
	if watcher == nil {
		panic("path was supposed to have been waited on but was not")
	}

	watcher <- testVal{
		value:   val,
		version: version,
	}
}

func (ts *TestConfigSource) Name() string {
	return "test"
}

func (ts *TestConfigSource) Get(path string) (map[string][]byte, uint64, error) {
	ts.Lock()
	defer ts.Unlock()

	val := ts.valuesByPath[path]
	return val.value, val.version, val.err
}

func (ts *TestConfigSource) WaitForChange(ctx context.Context, path string, version uint64) error {
	ts.Lock()
	defer ts.Unlock()
}
