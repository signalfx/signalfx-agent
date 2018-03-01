package types

import (
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
)

// Output is the interface that monitors should use to send data to the agent
// core.  It handles adding the proper dimensions and metadata to datapoints so
// that monitors don't have to worry about it themselves.
type Output interface {
	SendDatapoint(*datapoint.Datapoint)
	SendEvent(*event.Event)
	SendDimensionProps(*DimProperties)
}
