package sources

import (
	"context"
	"testing"
)

type testVal struct {
	value ma[string][]byte
	version uint64
	err error
}

type testSource struct {
	valuesByPath map[string]testVal
	valueCh chan testVal
}

func (ts *testSource) Get(path string) (map[string][]byte, uint64, error) {
	val := ts.values[path]
	return val.value, val.version, val.err
}

func (ts *testSource) WaitForChange(ctx context.Context, path string, version uint64) error {
	
}

func TestConfigSourceCacher(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	notices := make(chan string)
	testSource := newTestSource()

	cacher := newConfigSourceCacher(ctx, testSource, notices, true)
}
