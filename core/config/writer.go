package config

import (
	"net/url"

	"github.com/signalfx/neo-agent/core/filters"
)

// WriterConfig holds configuration for the datapoint writer.
type WriterConfig struct {
	// These are soft limits and affect how much memory will be initially
	// allocated for datapoints, not the maximum memory allowed.
	DatapointBufferCapacity      uint `yaml:"datapointBufferCapacity" default:"1000"`
	EventBufferCapacity          uint `yaml:"eventBufferCapacity" default:"1000"`
	DatapointSendIntervalSeconds int  `yaml:"datapointSendIntervalSeconds" default:"5"`
	EventSendIntervalSeconds     int  `yaml:"eventSendIntervalSeconds" default:"5"`
	// The following are propagated from the top level config
	IngestURL           *url.URL           `yaml:"-"`
	SignalFxAccessToken string             `yaml:"-"`
	GlobalDimensions    map[string]string  `yaml:"-"`
	Filter              *filters.FilterSet `yaml:"-"`
}
