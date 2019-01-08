package sampling

import (
	"context"

	"github.com/signalfx/gateway/etcdIntf"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/log"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/golib/trace"
)

// SmartSampleConfig is not here
type SmartSampleConfig struct {
	EtcdServer           etcdIntf.Server   `json:"-"`
	EtcdClient           etcdIntf.Client   `json:"-"`
	AdditionalDimensions map[string]string `json:",omitempty"`
}

// SmartSampler is not here
type SmartSampler struct{}

// StartupFinished does nothing
func (f *SmartSampler) StartupFinished() error {
	return nil
}

// AddSpans does nothing
func (f *SmartSampler) AddSpans(context context.Context, spans []*trace.Span, sink trace.Sink) error {
	return nil
}

// Datapoints adheres to the sfxclient.Collector interface
func (f *SmartSampler) Datapoints() []*datapoint.Datapoint {
	return []*datapoint.Datapoint{}
}

// Close does nothing
func (f *SmartSampler) Close() error {
	return nil
}

type dtsink interface {
	sfxclient.Sink
	trace.Sink
}

// ConfigureHTTPSink does nothing
func (f *SmartSampler) ConfigureHTTPSink(sink *sfxclient.HTTPSink) {
}

// New returns you nothing
func New(*SmartSampleConfig, log.Logger, dtsink) (*SmartSampler, error) {
	return nil, nil
}
