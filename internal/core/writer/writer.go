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
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/golib/trace"
	"github.com/signalfx/signalfx-agent/internal/core/common/dpmeta"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

const (
	// There cannot be more than this many events queued to be sent at any
	// given time.  This should be big enough for any reasonable use case.
	eventBufferCapacity = 1000
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
	spanChan     chan *trace.Span
	propertyChan chan *types.DimProperties

	ctx    context.Context
	cancel context.CancelFunc
	conf   *config.WriterConfig

	// map that holds host-specific ids like AWSUniqueID
	hostIDDims map[string]string

	dpBufferPool   *sync.Pool
	spanBufferPool *sync.Pool
	eventBuffer    []*event.Event

	dpRequestsActive        int64
	dpsInFlight             int64
	dpsSent                 int64
	traceSpanRequestsActive int64
	traceSpansInFlight      int64
	traceSpansSent          int64
	eventsSent              int64
	startTime               time.Time
}

// New creates a new un-configured writer
func New(conf *config.WriterConfig, dpChan chan *datapoint.Datapoint, eventChan chan *event.Event,
	propertyChan chan *types.DimProperties, spanChan chan *trace.Span) (*SignalFxWriter, error) {

	sw := &SignalFxWriter{
		conf:          conf,
		client:        sfxclient.NewHTTPSink(),
		dimPropClient: newDimensionPropertyClient(conf),
		hostIDDims:    conf.HostIDDims,
		dpChan:        dpChan,
		eventChan:     eventChan,
		spanChan:      spanChan,
		propertyChan:  propertyChan,
		startTime:     time.Now(),
		dpBufferPool: &sync.Pool{
			New: func() interface{} {
				return make([]*datapoint.Datapoint, 0, conf.DatapointMaxBatchSize)
			},
		},
		spanBufferPool: &sync.Pool{
			New: func() interface{} {
				return make([]*trace.Span, 0, conf.TraceSpanMaxBatchSize)
			},
		},
	}
	sw.ctx, sw.cancel = context.WithCancel(context.Background())

	sw.client.AuthToken = conf.SignalFxAccessToken

	sw.client.Client.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 90 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: conf.MaxRequests,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	dpEndpointURL, err := conf.IngestURL.Parse("v2/datapoint")
	if err != nil {
		log.WithFields(log.Fields{
			"error":     err,
			"ingestURL": conf.IngestURL.String(),
		}).Error("Could not construct datapoint ingest URL")
		return nil, err
	}
	sw.client.DatapointEndpoint = dpEndpointURL.String()

	eventEndpointURL, err := conf.IngestURL.Parse("v2/event")
	if err != nil {
		log.WithFields(log.Fields{
			"error":     err,
			"ingestURL": conf.IngestURL.String(),
		}).Error("Could not construct event ingest URL")
		return nil, err
	}
	sw.client.EventEndpoint = eventEndpointURL.String()

	traceEndpointURL := conf.TraceEndpointURL
	if traceEndpointURL == nil {
		var err error
		traceEndpointURL, err = conf.IngestURL.Parse("v1/trace")
		if err != nil {
			log.WithFields(log.Fields{
				"error":     err,
				"ingestURL": conf.IngestURL.String(),
			}).Error("Could not construct trace ingest URL")
			return nil, err
		}
	}
	sw.client.TraceEndpoint = traceEndpointURL.String()

	go sw.listenForDatapoints()
	go sw.listenForEventsAndDimProps()
	go sw.listenForTraceSpans()

	return sw, nil
}

func (sw *SignalFxWriter) shouldSendDatapoint(dp *datapoint.Datapoint) bool {
	return sw.conf.Filter == nil || !sw.conf.Filter.Matches(dp)
}

func (sw *SignalFxWriter) preprocessDatapoint(dp *datapoint.Datapoint) {
	dp.Dimensions = sw.addGlobalDims(dp.Dimensions)

	// Some metrics aren't really specific to the host they are running
	// on and shouldn't have any host-specific dims
	if b, ok := dp.Meta[dpmeta.NotHostSpecificMeta].(bool); !ok || !b {
		dp.Dimensions = sw.addhostIDFields(dp.Dimensions)
	}

	if sw.conf.LogDatapoints {
		log.Debugf("Sending datapoint:\n%s", utils.DatapointToString(dp))
	}
}

func (sw *SignalFxWriter) sendDatapoints(dps []*datapoint.Datapoint) error {
	// This sends synchonously
	err := sw.client.AddDatapoints(context.Background(), dps)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Error shipping datapoints to SignalFx")
		// If there is an error sending datapoints then just forget about them.
		return err
	}
	atomic.AddInt64(&sw.dpsSent, int64(len(dps)))
	log.Debugf("Sent %d datapoints to SignalFx", len(dps))

	return nil
}

func (sw *SignalFxWriter) sendEvents(events []*event.Event) error {
	for i := range events {
		events[i].Dimensions = sw.addGlobalDims(events[i].Dimensions)

		ps := events[i].Properties
		var notHostSpecific bool
		if ps != nil {
			if b, ok := ps[dpmeta.NotHostSpecificMeta].(bool); ok {
				notHostSpecific = b
				// Clear this so it doesn't leak through to ingest
				delete(ps, dpmeta.NotHostSpecificMeta)
			}
		}
		// Only override host dimension for now and omit other host id dims.
		if !notHostSpecific && sw.hostIDDims != nil && sw.hostIDDims["host"] != "" {
			events[i].Dimensions["host"] = sw.hostIDDims["host"]
		}

		if sw.conf.LogEvents {
			log.WithFields(log.Fields{
				"event": spew.Sdump(events[i]),
			}).Debug("Sending event")
		}
	}

	err := sw.client.AddEvents(context.Background(), events)
	if err != nil {
		log.WithError(err).Error("Error shipping events to SignalFx")
		return err
	}
	sw.eventsSent += int64(len(events))
	log.Debugf("Sent %d events to SignalFx", len(events))

	return nil
}

// Mutates datapoint dimensions in place to add global dimensions.  Also
// returns dims in case they were nil to begin with, so the return value should
// be assigned back to the dp Dimensions field.
func (sw *SignalFxWriter) addGlobalDims(dims map[string]string) map[string]string {
	if dims == nil {
		dims = make(map[string]string)
	}
	for name, value := range sw.conf.GlobalDimensions {
		// If the dimensions are already set, don't override
		if _, ok := dims[name]; !ok {
			dims[name] = value
		}
	}
	return dims
}

// Adds the host ids to the given map (e.g. dimensions/span tags), forcibly
// overridding any existing fields of the same name.
func (sw *SignalFxWriter) addhostIDFields(fields map[string]string) map[string]string {
	if fields == nil {
		fields = make(map[string]string)
	}
	for k, v := range sw.hostIDDims {
		fields[k] = v
	}
	return fields
}

// listenForDatapoints waits for datapoints to come in on the provided
// channels and forwards them to SignalFx.
func (sw *SignalFxWriter) listenForDatapoints() {

	// This acts like a semaphore if a request goroutine pushes into it before
	// making the request and pulling out of it when the request is done.
	// Pushes will block if there are more than the max number of outstanding
	// requests.
	dpSema := make(chan struct{}, sw.conf.DatapointMaxRequests)

	for {
		select {
		case <-sw.ctx.Done():
			return

		case dp := <-sw.dpChan:
			if !sw.shouldSendDatapoint(dp) {
				continue
			}
			buf := append(sw.dpBufferPool.Get().([]*datapoint.Datapoint), dp)
			buf = sw.drainDpChan(buf)

			for i := range buf {
				sw.preprocessDatapoint(buf[i])
			}

			atomic.AddInt64(&sw.dpsInFlight, int64(len(buf)))

			go func() {
				// Wait if there are more than the max outstanding requests
				dpSema <- struct{}{}

				atomic.AddInt64(&sw.dpRequestsActive, 1)
				sw.sendDatapoints(buf)

				<-dpSema

				atomic.AddInt64(&sw.dpRequestsActive, -1)
				atomic.AddInt64(&sw.dpsInFlight, -int64(len(buf)))

				sw.dpBufferPool.Put(buf[:0])
			}()
		}

	}
}

func (sw *SignalFxWriter) drainDpChan(buf []*datapoint.Datapoint) []*datapoint.Datapoint {
	for {
		select {
		case dp := <-sw.dpChan:
			if !sw.shouldSendDatapoint(dp) {
				continue
			}

			buf = append(buf, dp)
			if len(buf) >= sw.conf.DatapointMaxBatchSize {
				return buf
			}
		default:
			return buf
		}
	}
}

func (sw *SignalFxWriter) listenForEventsAndDimProps() {
	eventTicker := time.NewTicker(time.Duration(sw.conf.EventSendIntervalSeconds) * time.Second)
	defer eventTicker.Stop()

	initEventBuffer := func() {
		sw.eventBuffer = make([]*event.Event, 0, eventBufferCapacity)
	}
	initEventBuffer()

	for {
		select {
		case <-sw.ctx.Done():
			return

		case event := <-sw.eventChan:
			if len(sw.eventBuffer) > eventBufferCapacity {
				log.WithFields(log.Fields{
					"eventType":         event.EventType,
					"eventBufferLength": len(sw.eventBuffer),
				}).Error("Dropping event due to overfull buffer")
				continue
			}
			sw.eventBuffer = append(sw.eventBuffer, event)

		case <-eventTicker.C:
			if len(sw.eventBuffer) > 0 {
				go sw.sendEvents(sw.eventBuffer)
				initEventBuffer()
			}
		case dimProps := <-sw.propertyChan:
			// Run the sync async so we don't block other cases in this select
			go func(innerProps *types.DimProperties) {
				err := sw.dimPropClient.SetPropertiesOnDimension(innerProps)
				if err != nil {
					log.WithFields(log.Fields{
						"error":    err,
						"dimProps": innerProps,
					}).Error("Could not sync properties to dimension")
				}
			}(dimProps)
		}
	}
}

// Shutdown the writer and stop sending datapoints
func (sw *SignalFxWriter) Shutdown() {
	if sw.cancel != nil {
		sw.cancel()
	}
	log.Debug("Stopped datapoint writer")
}
