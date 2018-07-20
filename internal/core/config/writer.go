package config

import (
	"net/url"

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
	// The maximum number of concurrent requests to make to the ingest server
	// with datapoints.  This number multipled by `datapointMaxBatchSize` is
	// more or less the maximum number of datapoints that can be "in-flight" at
	// any given time.
	DatapointMaxRequests int `yaml:"datapointMaxRequests" default:"10"`
	// The agent does not send events immediately upon a monitor generating
	// them, but buffers them and sends them in batches.  The lower this
	// number, the less delay for events to appear in SignalFx.
	EventSendIntervalSeconds int `yaml:"eventSendIntervalSeconds" default:"1"`
	// The analogue of `datapointMaxRequests` for dimension property requests.
	PropertiesMaxRequests uint `yaml:"propertiesMaxRequests" default:"10"`
	// Properties that are synced to SignalFx are cached to prevent duplicate
	// requests from being sent, causing unnecessary load on our backend.
	PropertiesHistorySize uint `yaml:"propertiesHistorySize" default:"1000"`
	// If the log level is set to `debug` and this is true, all datapoints
	// generated by the agent will be logged.
	LogDatapoints bool `yaml:"logDatapoints"`
	// The analogue of `logDatapoints` for events.
	LogEvents bool `yaml:"logEvents"`
	// The following are propagated from elsewhere
	HostIDDims          map[string]string    `yaml:"-"`
	IngestURL           *url.URL             `yaml:"-"`
	APIURL              *url.URL             `yaml:"-"`
	SignalFxAccessToken string               `yaml:"-"`
	GlobalDimensions    map[string]string    `yaml:"-"`
	Filter              *dpfilters.FilterSet `yaml:"-"`
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
