package monitors

import (
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/golib/trace"
	"github.com/signalfx/signalfx-agent/internal/core/common/dpmeta"
	"github.com/signalfx/signalfx-agent/internal/core/services"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

// The default implementation of Output
type monitorOutput struct {
	*monitorFiltering
	monitorType               string
	monitorID                 types.MonitorID
	notHostSpecific           bool
	disableEndpointDimensions bool
	configHash                uint64
	endpoint                  services.Endpoint
	dpChan                    chan<- *datapoint.Datapoint
	eventChan                 chan<- *event.Event
	spanChan                  chan<- *trace.Span
	dimPropChan               chan<- *types.DimProperties
	extraDims                 map[string]string
}

var _ types.Output = &monitorOutput{}

// Copy the output so that you can attach a different set of dimensions to it.
func (mo *monitorOutput) Copy() types.Output {
	o := *mo
	o.extraDims = utils.CloneStringMap(mo.extraDims)
	o.filterSet = &(*mo.filterSet)
	return &o
}

func (mo *monitorOutput) SendDatapoint(dp *datapoint.Datapoint) {
	if dp.Meta == nil {
		dp.Meta = map[interface{}]interface{}{}
	}

	dp.Meta[dpmeta.MonitorIDMeta] = mo.monitorID
	dp.Meta[dpmeta.MonitorTypeMeta] = mo.monitorType
	dp.Meta[dpmeta.ConfigHashMeta] = mo.configHash
	if mo.notHostSpecific {
		dp.Meta[dpmeta.NotHostSpecificMeta] = true
	}

	var endpointDims map[string]string
	if mo.endpoint != nil && !mo.disableEndpointDimensions {
		endpointDims = mo.endpoint.Dimensions()
		dp.Meta[dpmeta.EndpointIDMeta] = mo.endpoint.Core().ID
	}

	dp.Dimensions = utils.MergeStringMaps(dp.Dimensions, mo.extraDims, endpointDims)
	// Defer filtering until here so we have the full dimension set to match
	// on.
	if mo.monitorFiltering != nil && mo.monitorFiltering.filterSet.Matches(dp) {
		return
	}

	mo.dpChan <- dp
}

func (mo *monitorOutput) SendEvent(event *event.Event) {
	if mo.notHostSpecific {
		if event.Properties == nil {
			event.Properties = make(map[string]interface{})
		}
		// Events don't have a non-serialized meta field, so just use
		// properties and make sure to remove this in the writer.
		event.Properties[dpmeta.NotHostSpecificMeta] = true
	}
	mo.eventChan <- event
}

func (mo *monitorOutput) SendSpan(span *trace.Span) {
	mo.spanChan <- span
}

func (mo *monitorOutput) SendDimensionProps(dimProps *types.DimProperties) {
	mo.dimPropChan <- dimProps
}

// AddExtraDimension can be called by monitors *before* datapoints are flowing
// to add an extra dimension value to all datapoints coming out of this output.
// This method is not thread-safe!
func (mo *monitorOutput) AddExtraDimension(key, value string) {
	mo.extraDims[key] = value
}

// RemoveExtraDimension will remove any dimension added to this output, either
// from the original configuration or from the AddExtraDimensions method.
// This method is not thread-safe!
func (mo *monitorOutput) RemoveExtraDimension(key string) {
	delete(mo.extraDims, key)
}
