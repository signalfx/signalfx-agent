package config

import (
	"net/url"

	"github.com/mitchellh/hashstructure"
	"github.com/signalfx/neo-agent/core/filters"
	log "github.com/sirupsen/logrus"
)

// WriterConfig holds configuration for the datapoint writer.
type WriterConfig struct {
	// These are soft limits and affect how much memory will be initially
	// allocated for datapoints, not the maximum memory allowed.
	// Both capacity options get applied at startup and subsequent changes
	// require an agent restart.
	DatapointBufferCapacity      uint `yaml:"datapointBufferCapacity" default:"1000"`
	EventBufferCapacity          uint `yaml:"eventBufferCapacity" default:"1000"`
	DatapointSendIntervalSeconds int  `yaml:"datapointSendIntervalSeconds" default:"5"`
	EventSendIntervalSeconds     int  `yaml:"eventSendIntervalSeconds" default:"5"`
	LogDatapoints                bool `yaml:"logDatapoints"`
	LogEvents                    bool `yaml:"logEvents"`
	// The following are propagated from the top level config
	IngestURL           *url.URL           `yaml:"-"`
	APIURL              *url.URL           `yaml:"-"`
	SignalFxAccessToken string             `yaml:"-"`
	GlobalDimensions    map[string]string  `yaml:"-"`
	Filter              *filters.FilterSet `yaml:"-"`
}

func (wc *WriterConfig) Hash() uint64 {
	hash, err := hashstructure.Hash(wc, nil)
	if err != nil {
		log.WithError(err).Error("Could not get hash of MonitorConfig struct")
		return 0
	}
	return hash
}
