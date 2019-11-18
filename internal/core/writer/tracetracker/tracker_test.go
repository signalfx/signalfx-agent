package tracetracker

import (
	"context"
	"testing"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/pointer"
	"github.com/signalfx/golib/v3/trace"
	"github.com/signalfx/signalfx-agent/internal/neotest"
	"github.com/stretchr/testify/assert"
)

func setTime(a *ActiveServiceTracker, t time.Time) {
	a.timeNow = neotest.PinnedNow(t)
}

func advanceTime(a *ActiveServiceTracker, minutes int64) {
	a.timeNow = neotest.AdvancedNow(a.timeNow, time.Duration(minutes)*time.Minute)
}

func TestDatapointsAreGenerated(t *testing.T) {
	a := New(5*time.Minute, nil)

	a.AddSpans(context.Background(), []*trace.Span{
		{
			LocalEndpoint: &trace.Endpoint{
				ServiceName: pointer.String("one"),
			},
		},
		{
			LocalEndpoint: &trace.Endpoint{
				ServiceName: pointer.String("two"),
			},
		},
	})

	dps := a.CorrelationDatapoints()
	assert.Len(t, dps, 2, "Expected two datapoints")

	var serviceDims []string
	for _, dp := range dps {
		serviceDims = append(serviceDims, dp.Dimensions["sf_hasService"])
	}
	assert.ElementsMatch(t, serviceDims, []string{"one", "two"}, "expected service names 'one' and 'two'")

	assert.Equal(t, dps[0].Value.(datapoint.IntValue).Int(), int64(0), "Expected dp value to be 0")
}

func TestExpiration(t *testing.T) {
	a := New(5*time.Minute, nil)
	setTime(a, time.Unix(100, 0))

	a.AddSpans(context.Background(), []*trace.Span{
		{
			LocalEndpoint: &trace.Endpoint{
				ServiceName: pointer.String("one"),
			},
		},
		{
			LocalEndpoint: &trace.Endpoint{
				ServiceName: pointer.String("two"),
			},
		},
		{
			LocalEndpoint: &trace.Endpoint{
				ServiceName: pointer.String("three"),
			},
		},
	})

	assert.Equal(t, a.activeServiceCount, int64(3), "activeServiceCount is not properly tracked")

	advanceTime(a, 4)

	a.AddSpans(context.Background(), []*trace.Span{
		{
			LocalEndpoint: &trace.Endpoint{
				ServiceName: pointer.String("two"),
			},
		},
	})

	advanceTime(a, 2)

	dps := a.CorrelationDatapoints()
	assert.Len(t, dps, 1, "Expected one datapoint")
	assert.Equal(t, dps[0].Dimensions["sf_hasService"], "two", "expected service two to still be active")

	assert.Equal(t, a.activeServiceCount, int64(1), "activeServiceCount is not properly tracked")
	assert.Equal(t, a.purgedServiceCount, int64(2), "purgedServiceCount is not properly tracked")

	advanceTime(a, 4)
	assert.Len(t, a.CorrelationDatapoints(), 0, "Expected all datapoints to be expired")
}
