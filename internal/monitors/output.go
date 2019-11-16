package monitors

import (
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/event"
	"github.com/signalfx/golib/v3/trace"
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
	dpChan                    chan<- []*datapoint.Datapoint
	eventChan                 chan<- *event.Event
	spanChan                  chan<- []*trace.Span
	dimensionChan             chan<- *types.Dimension
	extraDims                 map[string]string
	dimensionTransformations  map[string]string
}

var _ types.Output = &monitorOutput{}

// Copy the output so that you can attach a different set of dimensions to it.
func (mo *monitorOutput) Copy() types.Output {
	o := *mo
	o.extraDims = utils.CloneStringMap(mo.extraDims)
	o.dimensionTransformations = utils.CloneStringMap(mo.dimensionTransformations)
	o.filterSet = &(*mo.filterSet)
	return &o
}

func (mo *monitorOutput) SendDatapoints(dps ...*datapoint.Datapoint) {
	// This is the filtering in place trick from https://github.com/golang/go/wiki/SliceTricks#filter-in-place
	n := 0
	for i := range dps {
		if mo.preprocessDP(dps[i]) {
			dps[n] = dps[i]
			n++
		}
	}

	if n > 0 {
		mo.dpChan <- dps[:n]
	}
}

func (mo *monitorOutput) preprocessDP(dp *datapoint.Datapoint) bool {
	if dp.Meta == nil {
		dp.Meta = map[interface{}]interface{}{}
	}

	dp.Meta[dpmeta.MonitorIDMeta] = mo.monitorID
	dp.Meta[dpmeta.MonitorTypeMeta] = mo.monitorType
	dp.Meta[dpmeta.ConfigHashMeta] = mo.configHash
	if mo.notHostSpecific {
		dp.Meta[dpmeta.NotHostSpecificMeta] = true
	}

	dp.Meta[dpmeta.EndpointMeta] = mo.endpoint

	var endpointDims map[string]string
	if mo.endpoint != nil && !mo.disableEndpointDimensions {
		endpointDims = mo.endpoint.Dimensions()
	}

	dp.Dimensions = utils.MergeStringMaps(dp.Dimensions, mo.extraDims, endpointDims)

	// Defer filtering until here so we have the full dimension set to match
	// on.
	if mo.monitorFiltering.filterSet.Matches(dp) {
		return false
	}

	for origName, newName := range mo.dimensionTransformations {
		if v, ok := dp.Dimensions[origName]; ok {
			// If the new name is not an empty string transform the dimension
			if len(newName) > 0 {
				dp.Dimensions[newName] = v
			}
			delete(dp.Dimensions, origName)
		}
	}

	return true
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

func (mo *monitorOutput) SendSpans(spans ...*trace.Span) {
	for i := range spans {
		spans[i].Meta[dpmeta.EndpointMeta] = mo.endpoint
	}

	mo.spanChan <- spans
}

func (mo *monitorOutput) SendDimensionUpdate(dimensions *types.Dimension) {
	mo.dimensionChan <- dimensions
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
