package jmx

import (
	"context"

	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

type Config struct {
}

type Monitor struct {
	Output types.Output
	conf   *Config
	ctx    context.Context
	cancel context.CancelFunc
}

func (m *Monitor) Configure(conf *Config) error {
}
