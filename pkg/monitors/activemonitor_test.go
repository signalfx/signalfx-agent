package monitors

import (
	"context"
	"testing"
	"time"

	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/stretchr/testify/require"
)

type collectMonitor struct {
	config.MonitorConfig
	called chan struct{}
}

func (c *collectMonitor) Configure(_ *Conf) error {
	return nil
}

func (c *collectMonitor) Collect(context.Context) {
	c.called <- struct{}{}
}

var _ Collectable = &collectMonitor{}

type Conf struct {
	config.MonitorConfig
}

func TestActiveMonitor_collectable(t *testing.T) {
	instance := &collectMonitor{
		MonitorConfig: config.MonitorConfig{
			Type:            "test-monitor",
			IntervalSeconds: 5,
		},
		called: make(chan struct{}),
	}
	am := &ActiveMonitor{
		instance:   instance,
		id:         "test-monitor",
		configHash: 0,
		output:     nil,
		config:     nil,
		endpoint:   nil,
		cancel:     nil,
	}

	Register(&Metadata{MonitorType: "test-monitor"}, func() interface{} {
		return instance
	}, &Conf{})

	defer DeregisterAll()

	conf := &Conf{
		MonitorConfig: instance.MonitorConfig,
	}

	require.NoError(t, am.configureMonitor(conf))

	select {
	case <-instance.called:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatalf("monitor was not collected within 2 seconds")
	}
}
