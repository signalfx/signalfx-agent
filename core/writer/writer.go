package writer

import (
	"context"
	"net/url"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/golib/sfxclient"
	log "github.com/sirupsen/logrus"
)

const writeInterval = 5 * time.Second
const eventSendInterval = 5 * time.Second

type SignalFxWriter struct {
	client      *sfxclient.HTTPSink
	dpChan      chan *datapoint.Datapoint
	eventChan   chan *event.Event
	stop        chan struct{}
	dpBuffer    []*datapoint.Datapoint
	eventBuffer []*event.Event
}

func New(ingestURL *url.URL, accessToken string) (*SignalFxWriter, error) {
	client := sfxclient.NewHTTPSink()

	client.AuthToken = accessToken

	endpointURL, err := ingestURL.Parse("v2/datapoint")
	if err != nil {
		log.WithFields(log.Fields{
			"error":     err,
			"ingestURL": ingestURL.String(),
		}).Error("Could not construct ingest URL")
		return nil, err
	}
	client.DatapointEndpoint = endpointURL.String()

	return &SignalFxWriter{
		client: client,
		// TODO: make channel buffer configurable
		dpChan:    make(chan *datapoint.Datapoint, 1000),
		eventChan: make(chan *event.Event, 1000),
		stop:      make(chan struct{}),
	}, nil
}

func (sw *SignalFxWriter) sendDatapoints(dps []*datapoint.Datapoint) error {
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
	return nil
}

func (sw *SignalFxWriter) DPChannel() chan<- *datapoint.Datapoint {
	return sw.dpChan
}

func (sw *SignalFxWriter) EventChannel() chan<- *event.Event {
	return sw.eventChan
}

func (sw *SignalFxWriter) ListenForDatapoints() {
	dpTicker := time.NewTicker(writeInterval)
	eventTicker := time.NewTicker(eventSendInterval)

	go func() {
		defer dpTicker.Stop()
		defer eventTicker.Stop()

		for {
			select {

			case <-sw.stop:
				return

			case dp := <-sw.dpChan:
				log.Debugf("Buffering datapoint: %s", dp.String())
				sw.dpBuffer = append(sw.dpBuffer, dp)

			case event := <-sw.eventChan:
				sw.eventBuffer = append(sw.eventBuffer, event)

				log.WithFields(log.Fields{
					"event": *event,
				}).Info("Event received")

			case <-dpTicker.C:
				if len(sw.dpBuffer) > 0 {
					sw.sendDatapoints(sw.dpBuffer)
					sw.dpBuffer = sw.dpBuffer[:0]
				}

			case <-eventTicker.C:
				if len(sw.eventBuffer) > 0 {
					// sw.sendEvents(sw.eventBuffer)
					sw.eventBuffer = sw.eventBuffer[:0]
				}
			}
		}
	}()
}

func (sw *SignalFxWriter) Shutdown() {
	sw.stop <- struct{}{}
}
