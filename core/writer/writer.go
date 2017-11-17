// Package writer contains the SignalFx writer.  The writer is responsible for
// sending datapoints and events to SignalFx ingest.  Ideally all data would
// flow through here, but right now a lot of it is written to ingest by
// collectd.
//
// The writer provides a channel that all monitors can submit datapoints on.
// All monitors should include the "monitorType" key in the `Meta` map of the
// datapoint for use in filtering.
package writer

import (
	"context"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/neo-agent/core/config"
	log "github.com/sirupsen/logrus"
)

type state int

const (
	stopped state = iota
	listening
)

// SignalFxWriter is what sends events and datapoints to SignalFx ingest.  It
// receives events/datapoints on two buffered channels and writes them to
// SignalFx on a regular interval.
type SignalFxWriter struct {
	client        *sfxclient.HTTPSink
	dimPropClient *dimensionPropertyClient
	// Monitors should send datapoints to this
	dpChan chan *datapoint.Datapoint
	// Monitors should send events to this
	eventChan    chan *event.Event
	propertyChan chan *DimProperties

	stopCh chan struct{}

	state state
	lock  sync.Mutex

	conf *config.WriterConfig

	dpBuffer    []*datapoint.Datapoint
	eventBuffer []*event.Event
	dpsSent     uint64
	eventsSent  uint64
}

// New creates a new un-configured writer
func New() *SignalFxWriter {
	return &SignalFxWriter{
		state:         stopped,
		stopCh:        make(chan struct{}),
		client:        sfxclient.NewHTTPSink(),
		dimPropClient: newDimensionPropertyClient(),
	}
}

// Configure configures and starts up a routine that writes any datapoints or
// events that come in on the exposed channels.
func (sw *SignalFxWriter) Configure(conf *config.WriterConfig) bool {
	sw.lock.Lock()
	defer sw.lock.Unlock()

	// The capacity configuration options are only set once on agent startup
	if sw.dpChan == nil {
		sw.dpChan = make(chan *datapoint.Datapoint, conf.DatapointBufferCapacity)
	}
	if sw.eventChan == nil {
		sw.eventChan = make(chan *event.Event, conf.EventBufferCapacity)
	}
	if sw.propertyChan == nil {
		sw.propertyChan = make(chan *DimProperties, 100)
	}

	sw.client.AuthToken = conf.SignalFxAccessToken
	sw.dimPropClient.Token = conf.SignalFxAccessToken

	endpointURL, err := conf.IngestURL.Parse("v2/datapoint")
	if err != nil {
		log.WithFields(log.Fields{
			"error":     err,
			"ingestURL": conf.IngestURL.String(),
		}).Error("Could not construct ingest URL")
		return false
	}
	sw.client.DatapointEndpoint = endpointURL.String()

	sw.conf = conf

	// Do a shutdown in case some of our config values changed
	sw.shutdownIfRunning()
	sw.ensureListeningForDatapoints()

	return true
}

func (sw *SignalFxWriter) filterAndSendDatapoints(dps []*datapoint.Datapoint) error {
	finalDps := make([]*datapoint.Datapoint, 0)
	for i := range dps {
		if sw.conf.Filter == nil || !sw.conf.Filter.Matches(dps[i]) {
			dps[i].Dimensions = sw.addGlobalDims(dps[i].Dimensions)
			finalDps = append(finalDps, dps[i])

			log.WithFields(log.Fields{
				"dp": spew.Sdump(dps[i]),
			}).Debug("Sending datapoint")
		}
	}

	// This sends synchonously despite what the first param might seem to
	// indicate
	err := sw.client.AddDatapoints(context.Background(), finalDps)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Error shipping datapoints to SignalFx")
		// If there is an error sending datapoints then just forget about them.
		return err
	}
	sw.dpsSent += uint64(len(finalDps))
	log.Debugf("Sent %d datapoints to SignalFx", len(finalDps))

	return nil
}

func (sw *SignalFxWriter) sendEvents(events []*event.Event) error {
	for i := range events {
		events[i].Dimensions = sw.addGlobalDims(events[i].Dimensions)
		log.WithFields(log.Fields{
			"event": spew.Sdump(events[i]),
		}).Debug("Sending event")
	}

	err := sw.client.AddEvents(context.Background(), events)
	if err != nil {
		log.WithError(err).Error("Error shipping events to SignalFx")
		return err
	}
	sw.eventsSent += uint64(len(events))
	log.Debugf("Sent %d events to SignalFx", len(events))

	return nil
}

// mutates datapoint in place to add global dimensions.  Also returns dims in
// case they were nil to begin with.
func (sw *SignalFxWriter) addGlobalDims(dims map[string]string) map[string]string {
	if dims == nil {
		dims = make(map[string]string)
	}
	for name, value := range sw.conf.GlobalDimensions {
		// If the dimensions is already set, don't override.
		if _, ok := dims[name]; !ok {
			dims[name] = value
		}
	}
	return dims
}

// DPChannel returns a channel that datapoints can be fed into that will be
// sent to SignalFx ingest.
func (sw *SignalFxWriter) DPChannel() chan<- *datapoint.Datapoint {
	if sw.dpChan == nil {
		panic("You must call Configure on the writer before getting the datapoint channel")
	}
	return sw.dpChan
}

// EventChannel returns a channel that events can be fed into that will be
// sent to SignalFx ingest.
func (sw *SignalFxWriter) EventChannel() chan<- *event.Event {
	if sw.dpChan == nil {
		panic("You must call Configure on the writer before getting the event channel")
	}
	return sw.eventChan
}

// DimPropertiesChannel returns a channel that datapoints can be fed into that will be
// sent to SignalFx ingest.
func (sw *SignalFxWriter) DimPropertiesChannel() chan<- *DimProperties {
	if sw.propertyChan == nil {
		panic("You must call Configure on the writer before getting the properties channel")
	}
	return sw.propertyChan
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
	dpTicker := time.NewTicker(time.Duration(sw.conf.DatapointSendIntervalSeconds) * time.Second)
	defer dpTicker.Stop()

	eventTicker := time.NewTicker(time.Duration(sw.conf.EventSendIntervalSeconds) * time.Second)
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
			go sw.filterAndSendDatapoints(sw.dpBuffer)
			go sw.sendEvents(sw.eventBuffer)

			close(sw.stopCh)
			return

		case dp := <-sw.dpChan:
			sw.dpBuffer = append(sw.dpBuffer, dp)
			// TODO: perhaps flush the buffer more frequently than the
			// dpSendInterval if we exceed the initial buffer capacity OR
			// dynamically increase the buffer capacity so we don't have to
			// resize it as often and risk `append` doing a copy.

		case event := <-sw.eventChan:
			sw.eventBuffer = append(sw.eventBuffer, event)

		case <-dpTicker.C:
			if len(sw.dpBuffer) > 0 {
				go sw.filterAndSendDatapoints(sw.dpBuffer)
				initDPBuffer()
			}

		case <-eventTicker.C:
			if len(sw.eventBuffer) > 0 {
				// TODO: actually send events to SignalFx
				go sw.sendEvents(sw.eventBuffer)
				initEventBuffer()
			}
		case dimProps := <-sw.propertyChan:
			err := sw.dimPropClient.SetPropertiesOnDimension(dimProps)
			if err != nil {
				log.WithFields(log.Fields{
					"error":    err,
					"dimProps": dimProps,
				}).Error("Could not sync properties to dimension")
			}
		}
	}
}

// Assumes lock if held when called
func (sw *SignalFxWriter) shutdownIfRunning() {
	if sw.state != stopped {
		sw.stopCh <- struct{}{}
		<-sw.stopCh
		sw.stopCh = make(chan struct{})

		sw.state = stopped
	}
}

// Shutdown the writer and stop sending datapoints
func (sw *SignalFxWriter) Shutdown() {
	sw.lock.Lock()
	defer sw.lock.Unlock()

	sw.shutdownIfRunning()
}
