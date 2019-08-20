package prometheusexporter

import (
	"context"
	"fmt"
	"github.com/prometheus/common/expfmt"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/sirupsen/logrus"
	"sync"

	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Monitor for prometheus exporter metrics
type Monitor struct {
	Output types.Output
	// Optional set of metric names that will be sent by default, all other
	// metrics derived from the exporter being dropped.
	IncludedMetrics map[string]bool
	// Extra dimensions to add in addition to those specified in the config.
	ExtraDimensions map[string]string
	// If true, IncludedMetrics is ignored and everything is sent.
	SendAll      bool
	Ctx          context.Context
	Cancel       func()
	client       *Client
	loggingEntry *logrus.Entry
	isConfigured bool
	configErr    error
	mux          sync.Mutex
}

// Configure the monitor and kick off volume metric syncing
func (m *Monitor) Configure(conf ConfigInterface) error {
	if m.configureOnceSync(conf); m.configErr == nil {
		m.readSendCloseAsync(conf)
	}
	return m.configErr
}

func (m *Monitor) configureOnceSync(conf ConfigInterface) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if !m.isConfigured {
		if m.Ctx == nil {
			fmt.Printf("configureOnceSync() m.Ctx %+v", m.Ctx)
			m.Ctx, m.Cancel = context.WithCancel(context.Background())
		}
		m.loggingEntry = logrus.WithFields(logrus.Fields{"monitorType": conf.GetMonitorType()})
		if m.client, m.configErr = conf.NewClient(); m.configErr != nil {
			m.loggingEntry.WithError(m.configErr).Error("Could not create prometheus client")
		}
		m.isConfigured = true
	}
}

func (m *Monitor) readSendCloseAsync(conf ConfigInterface) {
	utils.RunOnInterval(m.Ctx, func() {
		bodyReader, format, err := m.client.GetBodyReader()
		defer func() {
			if bodyReader != nil {
				bodyReader.Close()
			}
		}()
		if err != nil {
			m.loggingEntry.WithError(err).Error("Could not get prometheus metrics")
			return
		}
		decoder := expfmt.NewDecoder(bodyReader, format)
		var dps []*datapoint.Datapoint
		if dps, err = decodeMetrics(decoder); err != nil {
			m.loggingEntry.WithError(err).Error("Could not decode prometheus metrics from response body")
			return
		}
		dpSender := conf.NewDatapointSender()
		dpSender.SendDatapoints(m.Output, dps)
	}, conf.GetInterval())
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	fmt.Printf("Shutdown(): %+v", m.Ctx)
	if m.Cancel != nil {
		m.Cancel()
	}
}
