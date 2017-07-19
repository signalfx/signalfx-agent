package kubernetes

import (
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	"golang.org/x/net/context"
)

// SFXClient is a wrapper around sfxclient.HTTPSink to add support for global
// dimensions.  TODO: Implement this in the main go library
type SFXClient struct {
	*sfxclient.HTTPSink
	GlobalDims map[string]string
}

// NewSFXClient creates a new SignalFx client and accepts global dimensions
func NewSFXClient(globalDims map[string]string) *SFXClient {
	return &SFXClient{
		HTTPSink:   sfxclient.NewHTTPSink(),
		GlobalDims: globalDims,
	}
}

// AddDatapoints accepts and sends datapoints to ingest
func (s *SFXClient) AddDatapoints(ctx context.Context, points []*datapoint.Datapoint) (err error) {
	for _, dp := range points {
		dp.Dimensions = datapoint.AddMaps(dp.Dimensions, s.GlobalDims)
	}
	return s.HTTPSink.AddDatapoints(ctx, points)
}
