package config

import (
	"net/url"
	"time"

	"github.com/mitchellh/hashstructure"
	"github.com/signalfx/signalfx-agent/internal/core/dpfilters"
	log "github.com/sirupsen/logrus"
)

// WriterConfig holds configuration for the datapoint writer.
type WriterConfig struct {
	// The maximum number of datapoints to include in a batch before sending the
	// batch to the ingest server.  Smaller batch sizes than this will be sent
	// if datapoints originate in smaller chunks.
	DatapointMaxBatchSize int `yaml:"datapointMaxBatchSize" default:"1000"`
	// The analogue of `datapointMaxBatchSize` for trace spans.
	TraceSpanMaxBatchSize int `yaml:"traceSpanMaxBatchSize" default:"1000"`
	// Deprecated: use `maxRequests` instead.
	DatapointMaxRequests int `yaml:"datapointMaxRequests"`
	// The maximum number of concurrent requests to make to a single ingest server
	// with datapoints/events/trace spans.  This number multipled by
	// `datapointMaxBatchSize` is more or less the maximum number of datapoints
	// that can be "in-flight" at any given time.
	MaxRequests int `yaml:"maxRequests" default:"10"`
	// The agent does not send events immediately upon a monitor generating
	// them, but buffers them and sends them in batches.  The lower this
	// number, the less delay for events to appear in SignalFx.
	EventSendIntervalSeconds int `yaml:"eventSendIntervalSeconds" default:"1"`
	// The analogue of `maxRequests` for dimension property requests.
	PropertiesMaxRequests uint `yaml:"propertiesMaxRequests" default:"10"`
	// Properties that are synced to SignalFx are cached to prevent duplicate
	// requests from being sent, causing unnecessary load on our backend.
	PropertiesHistorySize uint `yaml:"propertiesHistorySize" default:"1000"`
	// If the log level is set to `debug` and this is true, all datapoints
	// generated by the agent will be logged.
	LogDatapoints bool `yaml:"logDatapoints"`
	// The analogue of `logDatapoints` for events.
	LogEvents bool `yaml:"logEvents"`
	// The analogue of `logDatapoints` for trace spans.
	LogTraceSpans bool `yaml:"logTraceSpans"`
	// Whether to send host correlation metrics to correlation traced services
	// with the underlying host
	SendTraceHostCorrelationMetrics *bool `yaml:"sendTraceHostCorrelationMetrics" default:"false"`
	// How long to wait after a trace span's service name is last seen to
	// continue sending the correlation datapoints for that service.  This
	// should be a duration string that is accepted by
	// https://golang.org/pkg/time/#ParseDuration.  This option is irrelvant if
	// `sendTraceHostCorrelationMetrics` is false.
	StaleServiceTimeout time.Duration `yaml:"staleServiceTimeout" default:"5m"`
	// How frequently to send host correlation metrics that are generated from
	// the service name seen in trace spans sent through or by the agent.  This
	// should be a duration string that is accepted by
	// https://golang.org/pkg/time/#ParseDuration.  This option is irrelvant if
	// `sendTraceHostCorrelationMetrics` is false.
	TraceHostCorrelationMetricsInterval time.Duration `yaml:"traceHostCorrelationMetricsInterval" default:"1m"`
	// The following are propagated from elsewhere
	HostIDDims          map[string]string    `yaml:"-"`
	IngestURL           *url.URL             `yaml:"-"`
	APIURL              *url.URL             `yaml:"-"`
	TraceEndpointURL    *url.URL             `yaml:"-"`
	SignalFxAccessToken string               `yaml:"-"`
	GlobalDimensions    map[string]string    `yaml:"-"`
	Filter              *dpfilters.FilterSet `yaml:"-"`
}

func (wc *WriterConfig) initialize() {
	if wc.DatapointMaxRequests != 0 {
		wc.MaxRequests = wc.DatapointMaxRequests
	} else {
		wc.DatapointMaxRequests = wc.MaxRequests
	}
}

// Hash calculates a unique hash value for this config struct
func (wc *WriterConfig) Hash() uint64 {
	hash, err := hashstructure.Hash(wc, nil)
	if err != nil {
		log.WithError(err).Error("Could not get hash of WriterConfig struct")
		return 0
	}
	return hash
}
