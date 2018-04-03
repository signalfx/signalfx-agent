// Package cadvisor contains a monitor that pulls cadvisor stats either
// directly from cadvisor or from the kubelet /stats endpoint that exposes
// cadvisor.
package cadvisor

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors/cadvisor/converter"
)

// Monitor pulls metrics from a cAdvisor-compatible endpoint
type Monitor struct {
	monConfig *config.MonitorConfig
	lock      sync.Mutex
	stop      chan bool
	stopped   chan bool
}

// Configure and start/restart cadvisor plugin
func (m *Monitor) Configure(monConfig *config.MonitorConfig, sendDP func(*datapoint.Datapoint), statProvider converter.InfoProvider) error {
	// Lock for reconfiguring the plugin
	m.lock.Lock()
	defer m.lock.Unlock()

	m.monConfig = monConfig

	m.stop = nil
	m.stopped = nil

	collector := converter.NewCadvisorCollector(statProvider, sendDP, monConfig.ExtraDimensions)

	m.stop, m.stopped = monitorNode(monConfig.IntervalSeconds, collector)

	return nil
}

func monitorNode(intervalSeconds int, collector *converter.CadvisorCollector) (stop chan bool, stopped chan bool) {
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	stop = make(chan bool, 1)
	stopped = make(chan bool, 1)

	go func() {
		collector.Collect()
		for {
			select {
			case <-stop:
				log.Info("Stopping cAdvisor collection")
				ticker.Stop()
				stopped <- true
				return
			case <-ticker.C:
				collector.Collect()
			}
		}
	}()

	return stop, stopped
}

// Shutdown cadvisor plugin
func (m *Monitor) Shutdown() {
	// tell cadvisor to stop
	if m.stop != nil {
		close(m.stop)
	}
}
