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
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/event"
	"github.com/signalfx/golib/v3/sfxclient"
	"github.com/signalfx/golib/v3/trace"
	"github.com/signalfx/signalfx-agent/internal/core/common/dpmeta"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/core/dpfilters"
	"github.com/signalfx/signalfx-agent/internal/core/writer/dimensions"
	"github.com/signalfx/signalfx-agent/internal/core/writer/tap"
	"github.com/signalfx/signalfx-agent/internal/core/writer/tracetracker"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	sfxwriter "github.com/signalfx/signalfx-go/writer"
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
	client          *sfxclient.HTTPSink
	dimensionClient *dimensions.DimensionClient
	datapointWriter *sfxwriter.DatapointWriter
	spanWriter      *sfxwriter.SpanWriter

	// Monitors should send events to this
	eventChan     chan *event.Event
	dimensionChan chan *types.Dimension

	ctx    context.Context
	cancel context.CancelFunc
	conf   *config.WriterConfig
	logger *utils.ThrottledLogger
	dpTap  *tap.DatapointTap

	// map that holds host-specific ids like AWSUniqueID
	hostIDDims       map[string]string
	datapointFilters *dpfilters.FilterSet

	eventBuffer []*event.Event

	// Keeps track of what service names have been seen in trace spans that are
	// emitted by the agent
	serviceTracker    *tracetracker.ActiveServiceTracker
	spanSourceTracker *tracetracker.SpanSourceTracker

	// Datapoints sent in the last minute
	datapointsLastMinute int64
	// Datapoints that tried to be sent but couldn't in the last minute
	datapointsFailedLastMinute int64
	// Events sent in the last minute
	eventsLastMinute int64
	// Spans sent in the last minute
	spansLastMinute int64

	dpChan            chan []*datapoint.Datapoint
	spanChan          chan []*trace.Span
	dpsFailedToSend   int64
	traceSpansDropped int64
	eventsSent        int64
	startTime         time.Time
}

// New creates a new un-configured writer
func New(conf *config.WriterConfig, dpChan chan []*datapoint.Datapoint, eventChan chan *event.Event,
	dimensionChan chan *types.Dimension, spanChan chan []*trace.Span,
	spanSourceTracker *tracetracker.SpanSourceTracker) (*SignalFxWriter, error) {
	logger := utils.NewThrottledLogger(logrus.WithFields(log.Fields{"component": "writer"}), 20*time.Second)

	ctx, cancel := context.WithCancel(context.Background())

	dimensionClient, err := dimensions.NewDimensionClient(ctx, conf)
	if err != nil {
		cancel()
		return nil, err
	}

	sw := &SignalFxWriter{
		ctx:               ctx,
		cancel:            cancel,
		conf:              conf,
		logger:            logger,
		client:            sfxclient.NewHTTPSink(),
		dimensionClient:   dimensionClient,
		hostIDDims:        conf.HostIDDims,
		eventChan:         eventChan,
		dimensionChan:     dimensionChan,
		startTime:         time.Now(),
		spanSourceTracker: spanSourceTracker,
		dpChan:            dpChan,
		spanChan:          spanChan,
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

	sw.dimensionClient.Start()

	go sw.listenForEventsAndDimensionUpdates()

	sw.datapointWriter = &sfxwriter.DatapointWriter{
		PreprocessFunc: sw.preprocessDatapoint,
		SendFunc:       sw.sendDatapoints,
		OverwriteFunc: func() {
			sw.logger.ThrottledWarning(fmt.Sprintf("A datapoint was overwritten in the write buffer, please consider increasing the writer.maxDatapointsBuffered config option to something greater than %d", conf.MaxDatapointsBuffered))
		},
		MaxBatchSize: conf.DatapointMaxBatchSize,
		MaxRequests:  conf.MaxRequests,
		MaxBuffered:  conf.MaxDatapointsBuffered,
		InputChan:    sw.dpChan,
	}

	sw.datapointWriter.Start(ctx)

	// The only reason this is on the struct and not a local var is so we can
	// easily get diagnostic metrics from it
	sw.serviceTracker = sw.startGeneratingHostCorrelationMetrics()

	sw.spanWriter = &sfxwriter.SpanWriter{
		PreprocessFunc: sw.preprocessSpan,
		SendFunc:       sw.sendSpans,
		MaxBatchSize:   conf.TraceSpanMaxBatchSize,
		MaxRequests:    conf.MaxRequests,
		MaxBuffered:    int(conf.MaxTraceSpansInFlight),
		InputChan:      sw.spanChan,
	}
	sw.spanWriter.Start(ctx)

	log.Infof("Sending datapoints to %s", sw.client.DatapointEndpoint)
	log.Infof("Sending trace spans to %s", sw.client.TraceEndpoint)

	return sw, nil
}

func (sw *SignalFxWriter) shouldSendDatapoint(dp *datapoint.Datapoint) bool {
	return sw.datapointFilters == nil || !sw.datapointFilters.Matches(dp)
}

func (sw *SignalFxWriter) preprocessDatapoint(dp *datapoint.Datapoint) bool {
	if !sw.shouldSendDatapoint(dp) {
		return false
	}

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

	return true
}

func (sw *SignalFxWriter) sendDatapoints(ctx context.Context, dps []*datapoint.Datapoint) error {
	// This sends synchonously
	err := sw.client.AddDatapoints(ctx, dps)
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

func (sw *SignalFxWriter) listenForEventsAndDimensionUpdates() {
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
		case dim := <-sw.dimensionChan:
			if err := sw.dimensionClient.AcceptDimension(dim); err != nil {
				log.WithFields(log.Fields{
					"dimName":  dim.Name,
					"dimValue": dim.Value,
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
