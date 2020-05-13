package kafka

import (
	"context"

	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/sirupsen/logrus"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for the postgresql monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`

	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`

	ClusterName string `yaml:"clusterName"`
}

// Monitor that collects postgresql stats
type Monitor struct {
	Output types.FilteringOutput
	ctx    context.Context
	cancel context.CancelFunc
}

// Configure the monitor and kick off metric collection
func (m *Monitor) Configure(conf *Config) error {
	m.ctx, m.cancel = context.WithCancel(context.Background())

	logger := logrus.WithFields(logrus.Fields{"monitorType": monitorMetadata.MonitorType})

	return nil
}
