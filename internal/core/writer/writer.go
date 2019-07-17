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
	"fmt"
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
	"github.com/signalfx/signalfx-agent/internal/core/dpfilters"
	"github.com/signalfx/signalfx-agent/internal/core/writer/properties"
	"github.com/signalfx/signalfx-agent/internal/core/writer/tap"
	"github.com/signalfx/signalfx-agent/internal/core/writer/tracetracker"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/sirupsen/logrus"
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
	dimPropClient *properties.DimensionPropertyClient

	// Monitors should send datapoints to this
	dpChan chan *datapoint.Datapoint
	// Monitors should send events to this
	eventChan    chan *event.Event
	spanChan     chan *trace.Span
	propertyChan chan *types.DimProperties

	ctx    context.Context
	cancel context.CancelFunc
	conf   *config.WriterConfig
	logger *utils.ThrottledLogger
	dpTap  *tap.DatapointTap

	// map that holds host-specific ids like AWSUniqueID
	hostIDDims       map[string]string
	datapointFilters *dpfilters.FilterSet

	dpBufferPool   *sync.Pool
	spanBufferPool *sync.Pool
	eventBuffer    []*event.Event

	// Keeps track of what service names have been seen in trace spans that are
	// emitted by the agent
	serviceTracker *tracetracker.ActiveServiceTracker

	// Datapoints sent in the last minute
	datapointsLastMinute int64
	// Events sent in the last minute
	eventsLastMinute int64
	// Spans sent in the last minute
	spansLastMinute int64

	dpRequestsActive        int64
	dpRequestsWaiting       int64
	dpsInFlight             int64
	dpsWaiting              int64
	dpsSent                 int64
	dpsReceived             int64
	dpsFiltered             int64
	traceSpanRequestsActive int64
	traceSpansInFlight      int64
	traceSpansSent          int64
	traceSpansDropped       int64
	traceSpansFailedToSend  int64
	eventsSent              int64
	startTime               time.Time
}

// New creates a new un-configured writer
func New(conf *config.WriterConfig, dpChan chan *datapoint.Datapoint, eventChan chan *event.Event,
	propertyChan chan *types.DimProperties, spanChan chan *trace.Span) (*SignalFxWriter, error) {
	logger := utils.NewThrottledLogger(logrus.WithFields(log.Fields{"component": "writer"}), 20*time.Second)

	ctx, cancel := context.WithCancel(context.Background())

	dimPropClient, err := properties.NewDimensionPropertyClient(ctx, conf)
	if err != nil {
		cancel()
		return nil, err
	}

	sw := &SignalFxWriter{
		ctx:           ctx,
		cancel:        cancel,
		conf:          conf,
		logger:        logger,
		client:        sfxclient.NewHTTPSink(),
		dimPropClient: dimPropClient,
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
				buf := make([]*trace.Span, 0, conf.TraceSpanMaxBatchSize)
				return &buf
			},
		},
	}
	go sw.maintainLastMinuteActivity()

	sw.client.AuthToken = conf.SignalFxAccessToken

	sw.client.Client.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: conf.MaxRequests,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	dpEndpointURL, err := conf.ParsedIngestURL().Parse("v2/datapoint")
	if err != nil {
		log.WithFields(log.Fields{
			"error":     err,
			"ingestURL": conf.ParsedIngestURL().String(),
		}).Error("Could not construct datapoint ingest URL")
		return nil, err
	}
	sw.client.DatapointEndpoint = dpEndpointURL.String()

	eventEndpointURL, err := conf.ParsedIngestURL().Parse("v2/event")
	if err != nil {
		log.WithFields(log.Fields{
			"error":     err,
			"ingestURL": conf.ParsedIngestURL().String(),
		}).Error("Could not construct event ingest URL")
		return nil, err
	}
	sw.client.EventEndpoint = eventEndpointURL.String()

	traceEndpointURL := conf.ParsedTraceEndpointURL()
	if traceEndpointURL == nil {
		var err error
		traceEndpointURL, err = conf.ParsedIngestURL().Parse("v1/trace")
		if err != nil {
			log.WithFields(log.Fields{
				"error":     err,
				"ingestURL": conf.ParsedIngestURL().String(),
			}).Error("Could not construct trace ingest URL")
			return nil, err
		}
	}
	sw.client.TraceEndpoint = traceEndpointURL.String()

	sw.datapointFilters, err = sw.conf.DatapointFilters()
	if err != nil {
		return nil, err
	}

	sw.dimPropClient.Start()
	go sw.listenForDatapoints()
	go sw.listenForEventsAndDimProps()
	go sw.listenForTraceSpans()

	log.Infof("Sending datapoints to %s", sw.client.DatapointEndpoint)
	log.Infof("Sending trace spans to %s", sw.client.TraceEndpoint)

	return sw, nil
}

func (sw *SignalFxWriter) shouldSendDatapoint(dp *datapoint.Datapoint) bool {
	return sw.datapointFilters == nil || !sw.datapointFilters.Matches(dp)
}

func (sw *SignalFxWriter) preprocessDatapoint(dp *datapoint.Datapoint) {
	dp.Dimensions = sw.addGlobalDims(dp.Dimensions)

	// Some metrics aren't really specific to the host they are running
	// on and shouldn't have any host-specific dims
	if b, ok := dp.Meta[dpmeta.NotHostSpecificMeta].(bool); !ok || !b {
		dp.Dimensions = sw.addhostIDFields(dp.Dimensions)
	}

	utils.TruncateDimensionValuesInPlace(dp.Dimensions)

	if sw.conf.LogDatapoints {
		log.Debugf("Sending datapoint:\n%s", utils.DatapointToString(dp))
	}
}

func (sw *SignalFxWriter) sendDatapoints(dps []*datapoint.Datapoint) error {
	// This sends synchonously
	err := sw.client.AddDatapoints(sw.ctx, dps)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Error shipping datapoints to SignalFx")
		// If there is an error sending datapoints then just forget about them.
		return err
	}
	log.Debugf("Sent %d datapoints out of the agent", len(dps))

	// dpTap.Accept handles the receiver being nil
	sw.dpTap.Accept(dps)

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
	bufferSize := sw.conf.MaxDatapointsBuffered
	maxRequests := int64(sw.conf.DatapointMaxRequests) //nolint: staticcheck
	// Ring buffer of datapoints, initialized to its maximum length to avoid
	// reallocations.
	dpBuffer := make([]*datapoint.Datapoint, bufferSize)
	// The index that marks the end of the last chunk of datapoints that was
	// sent.  It is one greater than the actual index, to match the golang
	// slice high range.
	lastHighStarted := 0
	// The next index within the buffer that a datapoint should be added to.
	nextDatapointIdx := 0
	// Corresponds to nextDatapointIdx but is easier to work with without modulo
	batched := 0
	requestDoneCh := make(chan struct{}, maxRequests)

	// How many times around the ring buffer we have gone when putting
	// datapoints onto the buffer
	bufferedCircuits := int64(0)
	// How many times around the ring buffer we have gone when starting
	// requests
	startedCircuits := int64(0)

	targetHighStarted := func() int {
		if nextDatapointIdx < lastHighStarted {
			// Wrap around happened, just take what we have left until wrap
			// around so that we can take a single slice of it since slice
			// ranges can't wrap around.
			return bufferSize
		}

		return nextDatapointIdx
	}

	tryToSendBufferChunk := func(newHigh int) bool {
		if newHigh == lastHighStarted { // Nothing added
			return false
		}

		if sw.dpRequestsActive >= maxRequests {
			sw.dpRequestsWaiting++
			sw.dpsWaiting += int64(newHigh - lastHighStarted)
			log.Debugf("Max datapoint requests hit, %d requests will be combined", sw.dpRequestsWaiting)
			return false
		}

		sw.dpRequestsActive++
		go func(low, high int) {
			dpCount := int64(high - low)
			atomic.AddInt64(&sw.dpsInFlight, dpCount)

			log.Debugf("Sending dpBuffer[%d:%d]", low, high)
			_ = sw.sendDatapoints(dpBuffer[low:high])

			atomic.AddInt64(&sw.dpsInFlight, -dpCount)
			atomic.AddInt64(&sw.dpsSent, dpCount)

			requestDoneCh <- struct{}{}
		}(lastHighStarted, newHigh)

		lastHighStarted = newHigh
		if lastHighStarted == bufferSize { // Wrap back to 0
			lastHighStarted = 0
			startedCircuits++
		}

		batched = 0
		sw.dpRequestsWaiting = 0
		sw.dpsWaiting = 0
		return true
	}

	handleRequestDone := func() {
		sw.dpRequestsActive--
		if sw.dpRequestsWaiting > 0 {
			tryToSendBufferChunk(targetHighStarted())
		}
	}

	processDP := func(dp *datapoint.Datapoint) {
		if !sw.shouldSendDatapoint(dp) {
			sw.dpsFiltered++
			if sw.conf.LogDroppedDatapoints {
				log.Debugf("Dropping datapoint:\n%s", utils.DatapointToString(dp))
			}
			return
		}

		sw.dpsReceived++
		sw.preprocessDatapoint(dp)
		dpBuffer[nextDatapointIdx] = dp

		nextDatapointIdx++
		if nextDatapointIdx == bufferSize { // Wrap around the buffer
			nextDatapointIdx = 0
			bufferedCircuits++
		}

		if lastHighStarted < nextDatapointIdx && bufferedCircuits > startedCircuits {
			sw.logger.ThrottledWarning(fmt.Sprintf("Datapoint ring buffer overflowed, some datapoints were dropped. Set writer.maxDatapointsBuffered to something higher (currently %d)", bufferSize))
		}
		batched++

		if batched >= sw.conf.DatapointMaxBatchSize {
			tryToSendBufferChunk(targetHighStarted())
		}
	}

	for {
		select {
		case <-sw.ctx.Done():
			return

		case dp := <-sw.dpChan:
			processDP(dp)

		case <-requestDoneCh:
			handleRequestDone()

		default:
			newHigh := targetHighStarted()
			// Could be less if wrapped around
			if newHigh != lastHighStarted {
				tryToSendBufferChunk(newHigh)
			}

			select {
			case <-sw.ctx.Done():
				return

			case <-requestDoneCh:
				handleRequestDone()

			case dp := <-sw.dpChan:
				processDP(dp)
			}
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
				go func(buf []*event.Event) {
					if err := sw.sendEvents(buf); err != nil {

						log.WithError(err).Error("Error shipping events to SignalFx")
					}
				}(sw.eventBuffer)
				initEventBuffer()
			}
		case dimProps := <-sw.propertyChan:
			if err := sw.dimPropClient.AcceptDimProp(dimProps); err != nil {
				log.WithFields(log.Fields{
					"dimName":  dimProps.Dimension.Name,
					"dimValue": dimProps.Dimension.Value,
				}).WithError(err).Warn("Dropping dimension update")
			}
		}
	}
}

// SetTap allows you to set one datapoint tap at a time to inspect datapoints
// going out of the agent.
func (sw *SignalFxWriter) SetTap(dpTap *tap.DatapointTap) {
	sw.dpTap = dpTap
}

// Shutdown the writer and stop sending datapoints
func (sw *SignalFxWriter) Shutdown() {
	if sw.cancel != nil {
		sw.cancel()
	}
	log.Debug("Stopped datapoint writer")
}
