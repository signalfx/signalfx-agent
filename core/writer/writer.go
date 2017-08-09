// The writer is responsible for sending datapoints and events to SignalFx
// ingest.  Ideally all data would flow through here, but right now a lot of it
// is written to ingest by collectd.
// The writer provides a channel that all monitors can submit datapoints on.
// All monitors should include the "monitorType" key in the `Meta` map of the
// datapoint for use in filtering.
package writer

import (
	"context"
	"sync"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/neo-agent/core/config"
	log "github.com/sirupsen/logrus"
)

const (
	dpSendInterval    = 5 * time.Second
	eventSendInterval = 5 * time.Second
)

type state int

const (
	stopped state = iota
	listening
)

type SignalFxWriter struct {
	client *sfxclient.HTTPSink
	// Monitors should send datapoints to this
	dpChan chan *datapoint.Datapoint
	// Monitors should send events to this
	eventChan chan *event.Event

	stopCh chan struct{}

	state state
	lock  sync.Mutex

	conf *config.WriterConfig

	dpBuffer    []*datapoint.Datapoint
	eventBuffer []*event.Event
	dpsSent     uint64
	eventsSent  uint64
}

func New() *SignalFxWriter {
	return &SignalFxWriter{
		state:  stopped,
		stopCh: make(chan struct{}),
	}
}

func (sw *SignalFxWriter) Configure(conf *config.WriterConfig) bool {
	sw.lock.Lock()
	defer sw.lock.Unlock()

	sw.dpChan = make(chan *datapoint.Datapoint, conf.DatapointBufferCapacity)
	sw.eventChan = make(chan *event.Event, conf.EventBufferCapacity)

	client := sfxclient.NewHTTPSink()

	client.AuthToken = conf.SignalFxAccessToken

	endpointURL, err := conf.IngestURL.Parse("v2/datapoint")
	if err != nil {
		log.WithFields(log.Fields{
			"error":     err,
			"ingestURL": conf.IngestURL.String(),
		}).Error("Could not construct ingest URL")
		return false
	}
	client.DatapointEndpoint = endpointURL.String()

	sw.client = client
	sw.conf = conf

	sw.ensureListeningForDatapoints()

	return true
}

func (sw *SignalFxWriter) filterAndSendDatapoints(dps []*datapoint.Datapoint) error {
	finalDps := make([]*datapoint.Datapoint, 0, len(dps))
	for i := range dps {
		if !sw.conf.Filter.Matches(dps[i]) {
			sw.addGlobalDimsToDatapoint(dps[i])
			finalDps = append(finalDps, dps[i])
		}
	}

	// This sends synchonously despite what the first param might seem to
	// indicate
	err := sw.client.AddDatapoints(context.Background(), dps)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Error shipping datapoints to SignalFx")
		// If there is an error sending datapoints then just forget about them.
		return err
	}
	sw.dpsSent += uint64(len(dps))
	return nil
}

// mutates datapoint in place to add global dimensions
func (sw *SignalFxWriter) addGlobalDimsToDatapoint(dp *datapoint.Datapoint) {
	for name, value := range sw.conf.GlobalDimensions {
		// If the dimensions is already set, don't override.
		if _, ok := dp.Dimensions[name]; !ok {
			dp.Dimensions[name] = value
		}
	}
}

func (sw *SignalFxWriter) DPChannel() chan<- *datapoint.Datapoint {
	return sw.dpChan
}

func (sw *SignalFxWriter) EventChannel() chan<- *event.Event {
	return sw.eventChan
}

// ensureListeningForDatapoints will make sure the writer is accepting
// datapoints if it is not already.  This method is idempotent.
// ASSUMES LOCK IS HELD WHEN CALLED.
func (sw *SignalFxWriter) ensureListeningForDatapoints() {
	if sw.state != listening {
		go sw.listenForDatapoints()
		sw.state = listening
	}
}

// listenForDatapoints starts up a goroutine that waits for datapoints and
// events to come in on the provided channels.  That goroutine also sends data
// to ingest at regular intervals.
func (sw *SignalFxWriter) listenForDatapoints() {
	dpTicker := time.NewTicker(dpSendInterval)
	defer dpTicker.Stop()

	eventTicker := time.NewTicker(eventSendInterval)
	defer eventTicker.Stop()

	initDPBuffer := func() {
		sw.dpBuffer = make([]*datapoint.Datapoint, 0, sw.conf.DatapointBufferCapacity)
	}
	initDPBuffer()

	initEventBuffer := func() {
		sw.eventBuffer = make([]*event.Event, 0, sw.conf.EventBufferCapacity)
	}
	initEventBuffer()

	for {
		select {

		case <-sw.stopCh:
			close(sw.stopCh)
			return

		case dp := <-sw.dpChan:
			log.Debugf("Buffering datapoint: %s", dp.String())
			sw.dpBuffer = append(sw.dpBuffer, dp)
			// TODO: perhaps flush the buffer more frequently than the
			// dpSendInterval if we exceed the initial buffer capacity OR
			// dynamically increase the buffer capacity so we don't have to
			// resize it as often and risk `append` doing a copy.

		case event := <-sw.eventChan:
			sw.eventBuffer = append(sw.eventBuffer, event)

			log.WithFields(log.Fields{
				"event": *event,
			}).Info("Event received")

		case <-dpTicker.C:
			if len(sw.dpBuffer) > 0 {
				go sw.filterAndSendDatapoints(sw.dpBuffer)
				initDPBuffer()
			}

		case <-eventTicker.C:
			if len(sw.eventBuffer) > 0 {
				// TODO: actually send events to SignalFx
				// go sw.sendEvents(eventBuffer)
				initEventBuffer()
			}
		}
	}
}

func (sw *SignalFxWriter) Shutdown() {
	sw.lock.Lock()
	defer sw.lock.Unlock()

	if sw.state != stopped {
		sw.stopCh <- struct{}{}
		sw.state = stopped
	}
}
