package types

import (
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/golib/trace"
	"github.com/signalfx/signalfx-agent/internal/core/dpfilters"
)

// Output is the interface that monitors should use to send data to the agent
// core.  It handles adding the proper dimensions and metadata to datapoints so
// that monitors don't have to worry about it themselves.
type Output interface {
	Copy() Output
	SendDatapoint(*datapoint.Datapoint)
	SendEvent(*event.Event)
	SendSpan(*trace.Span)
	SendDimensionProps(*DimProperties)
	AddExtraDimension(key, value string)
	RemoveExtraDimension(key string)

	AddDatapointExclusionFilter(filter dpfilters.DatapointFilter)
	EnabledMetrics() []string
}
