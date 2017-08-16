package config

import (
	"net/url"

	"github.com/signalfx/neo-agent/core/filters"
)

type WriterConfig struct {
	// These are soft limits and affect how much memory will be initially
	// allocated for datapoints, not the maximum memory allowed.
	DatapointBufferCapacity uint `default:"1000"`
	EventBufferCapacity     uint `default:"1000"`
	// The following are propagated from the top level config
	IngestURL           *url.URL           `yaml:"-"`
	SignalFxAccessToken string             `yaml:"-"`
	GlobalDimensions    map[string]string  `yaml:"-"`
	Filter              *filters.FilterSet `yaml:"-"`
}
