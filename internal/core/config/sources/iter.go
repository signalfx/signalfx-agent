package sources

import (
	"context"

	"github.com/signalfx/signalfx-agent/internal/core/config/types"
)

// ConfigSourceIterator is a simple helper that makes getting content from
// config sources a bit simpler by only exposing a single method to use.
type ConfigSourceIterator struct {
	cs      types.ConfigSource
	path    string
	version *uint64
}

func NewConfigSourceIterator(cs types.ConfigSource, path string) *ConfigSourceIterator {
	return &ConfigSourceIterator{
		cs:   cs,
		path: path,
	}
}

func (i *ConfigSourceIterator) getAndUpdateVersion() (map[string][]byte, error) {
	content, version, err := i.cs.Get(i.path)
	if err != nil {
		return nil, err
	}
	i.version = &version
	return content, nil
}

func (i *ConfigSourceIterator) Next(ctx context.Context) (map[string][]byte, error) {
	if i.version == nil {
		return i.getAndUpdateVersion()
	}
	err := i.cs.WaitForChange(ctx, i.path, *i.version)
	if err != nil {
		return nil, err
	}
	return i.getAndUpdateVersion()
}
