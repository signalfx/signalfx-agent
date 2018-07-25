package neotest

import (
	"sync"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
)

// TestOutput can be used in place of the normal monitor outut to provide a
// simpler way of testing monitor output.
type TestOutput struct {
	dpChan      chan *datapoint.Datapoint
	eventChan   chan *event.Event
	dimPropChan chan *types.DimProperties

	// Use a lock since monitors are allowed to use output from multiple
	// threads.
	lock sync.Mutex
}

// NewTestOutput creates a new initialized TestOutput instance
func NewTestOutput() *TestOutput {
	return &TestOutput{
		dpChan:      make(chan *datapoint.Datapoint, 1000),
		eventChan:   make(chan *event.Event, 1000),
		dimPropChan: make(chan *types.DimProperties, 1000),
	}
}

// SendDatapoint accepts a datapoint and sticks it in a buffered queue
func (to *TestOutput) SendDatapoint(dp *datapoint.Datapoint) {
	to.dpChan <- dp
}

// SendEvent accepts an event and sticks it in a buffered queue
func (to *TestOutput) SendEvent(event *event.Event) {
	to.eventChan <- event
}

// SendDimensionProps accepts a dim prop update and sticks it in a buffered queue
func (to *TestOutput) SendDimensionProps(dimProps *types.DimProperties) {
	to.dimPropChan <- dimProps
}

// AddExtraDimension is a noop here
func (to *TestOutput) AddExtraDimension(key, value string) {}

// RemoveExtraDimension is a noop here
func (to *TestOutput) RemoveExtraDimension(key string) {}

// FlushDatapoints returns all of the datapoints injected into the channel so
// far.
func (to *TestOutput) FlushDatapoints() []*datapoint.Datapoint {
	var out []*datapoint.Datapoint
	for {
		select {
		case dp := <-to.dpChan:
			out = append(out, dp)
		default:
			return out
		}
	}
}

// FlushEvents returns all of the datapoints injected into the channel so
// far.
func (to *TestOutput) FlushEvents() []*event.Event {
	var out []*event.Event
	for {
		select {
		case event := <-to.eventChan:
			out = append(out, event)
		default:
			return out
		}
	}
}

// WaitForDPs will keep pulling datapoints off of the internal queue until it
// either gets the expected count or waitSeconds seconds have elapsed.  It then
// returns those datapoints.  It will never return more than 'count' datapoints.
func (to *TestOutput) WaitForDPs(count, waitSeconds int) []*datapoint.Datapoint {
	var dps []*datapoint.Datapoint
	timeout := time.After(time.Duration(waitSeconds) * time.Second)

loop:
	for {
		select {
		case dp := <-to.dpChan:
			dps = append(dps, dp)
			if len(dps) >= count {
				break loop
			}
		case <-timeout:
			break loop
		}
	}

	return dps
}

// WaitForDimensionProps will keep pulling dimension property updates off of
// the internal queue until it either gets the expected count or waitSeconds
// seconds have elapsed.  It then returns those dimension property updates.  It
// will never return more than 'count' objects.
func (to *TestOutput) WaitForDimensionProps(count, waitSeconds int) []*types.DimProperties {
	var dps []*types.DimProperties
	timeout := time.After(time.Duration(waitSeconds) * time.Second)

loop:
	for {
		select {
		case dp := <-to.dimPropChan:
			dps = append(dps, dp)
			if len(dps) >= count {
				break loop
			}
		case <-timeout:
			break loop
		}
	}

	return dps
}
