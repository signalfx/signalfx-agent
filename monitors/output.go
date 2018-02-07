package monitors

import (
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/neo-agent/core/common/dpmeta"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/monitors/types"
	"github.com/signalfx/neo-agent/utils"
)

// The default implementation of Output
type monitorOutput struct {
	monitorType string
	monitorID   types.MonitorID
	configHash  uint64
	endpoint    services.Endpoint
	dpChan      chan<- *datapoint.Datapoint
	eventChan   chan<- *event.Event
	dimPropChan chan<- *types.DimProperties
	extraDims   map[string]string
}

func (mo *monitorOutput) SendDatapoint(dp *datapoint.Datapoint) {
	if dp.Meta == nil {
		dp.Meta = make(map[interface{}]interface{})
	}

	dp.Meta[dpmeta.MonitorIDMeta] = mo.monitorID
	dp.Meta[dpmeta.MonitorTypeMeta] = mo.monitorType
	dp.Meta[dpmeta.ConfigHashMeta] = mo.configHash

	var endpointDims map[string]string
	if mo.endpoint != nil {
		endpointDims = mo.endpoint.Dimensions()
		dp.Meta[dpmeta.EndpointIDMeta] = mo.endpoint.Core().ID
	}

	dp.Dimensions = utils.MergeStringMaps(dp.Dimensions, mo.extraDims, endpointDims)

	mo.dpChan <- dp
}

func (mo *monitorOutput) SendEvent(event *event.Event) {
	mo.eventChan <- event
}

func (mo *monitorOutput) SendDimensionProps(dimProps *types.DimProperties) {
	mo.dimPropChan <- dimProps
}
