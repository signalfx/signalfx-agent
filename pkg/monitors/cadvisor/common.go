// Package cadvisor contains a monitor that pulls cadvisor stats either
// directly from cadvisor or from the kubelet /stats endpoint that exposes
// cadvisor.
package cadvisor

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors/cadvisor/converter"
)

// Monitor pulls metrics from a cAdvisor-compatible endpoint
type Monitor struct {
	monConfig *config.MonitorConfig
	stop      chan bool
}

// Configure and start/restart cadvisor plugin
func (m *Monitor) Configure(
	monConfig *config.MonitorConfig,
	sendDPs func(...*datapoint.Datapoint),
	statProvider converter.InfoProvider,
	hasPodEphemeralStorageStatsGroupEnabled bool) error {

	m.monConfig = monConfig

	collector := converter.NewCadvisorCollector(statProvider, sendDPs, monConfig.ExtraDimensions)

	m.stop = monitorNode(monConfig.IntervalSeconds, collector, hasPodEphemeralStorageStatsGroupEnabled)

	return nil
}

func monitorNode(intervalSeconds int, collector *converter.CadvisorCollector, hasPodEphemeralStorageStatsGroupEnabled bool) (stop chan bool) {
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	stop = make(chan bool, 1)

	go func() {
		collector.Collect(hasPodEphemeralStorageStatsGroupEnabled)
		for {
			select {
			case <-stop:
				log.Info("Stopping cAdvisor collection")
				ticker.Stop()
				return
			case <-ticker.C:
				collector.Collect(hasPodEphemeralStorageStatsGroupEnabled)
			}
		}
	}()

	return stop
}

// Shutdown cadvisor plugin
func (m *Monitor) Shutdown() {
	// tell cadvisor to stop
	if m.stop != nil {
		close(m.stop)
	}
}
