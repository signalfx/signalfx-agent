package monitors

import (
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/golib/trace"
	"github.com/signalfx/signalfx-agent/internal/core/common/dpmeta"
	"github.com/signalfx/signalfx-agent/internal/core/dpfilters"
	"github.com/signalfx/signalfx-agent/internal/core/services"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

// The default implementation of Output
type monitorOutput struct {
	monitorType               string
	monitorID                 types.MonitorID
	notHostSpecific           bool
	disableEndpointDimensions bool
	oldFilter                 *dpfilters.FilterSet
	newFilter                 *dpfilters.FilterSet
	configHash                uint64
	endpoint                  services.Endpoint
	dpChan                    chan<- *datapoint.Datapoint
	eventChan                 chan<- *event.Event
	spanChan                  chan<- *trace.Span
	dimPropChan               chan<- *types.DimProperties
	extraDims                 map[string]string
}

var _ types.Output = &monitorOutput{}

func (mo *monitorOutput) SendDatapoint(dp *datapoint.Datapoint) {
	if mo.oldFilter != nil && mo.oldFilter.Matches(dp) {
		return
	}
	if mo.newFilter != nil && mo.newFilter.Matches(dp) {
		return
	}

	if dp.Meta == nil {
		dp.Meta = make(map[interface{}]interface{})
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
