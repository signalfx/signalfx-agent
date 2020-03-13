package tracetracker

import (
	"context"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	"sync"
	"testing"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/pointer"
	"github.com/signalfx/golib/v3/trace"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/core/writer/correlations"
	"github.com/signalfx/signalfx-agent/pkg/neotest"
	"github.com/stretchr/testify/assert"
)

func setTime(a *ActiveServiceTracker, t time.Time) {
	a.timeNow = neotest.PinnedNow(t)
}

func advanceTime(a *ActiveServiceTracker, minutes int64) {
	a.timeNow = neotest.AdvancedNow(a.timeNow, time.Duration(minutes)*time.Minute)
}

func TestDatapointsAreGenerated(t *testing.T) {
	testCtx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()
	correlationClient, err := correlations.NewCorrelationClient(testCtx, &config.WriterConfig{})
	assert.NoError(t, err, "failed to create correlation client")

	a := New(5*time.Minute, correlationClient, nil, "", nil)

	a.AddSpans(context.Background(), []*trace.Span{
		{
			LocalEndpoint: &trace.Endpoint{
				ServiceName: pointer.String("one"),
			},
			Tags: map[string]string{"host": "test"},
		},
		{
			LocalEndpoint: &trace.Endpoint{
				ServiceName: pointer.String("two"),
			},
			Tags: map[string]string{"host": "test"},
		},
	})

	a.Purge()
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
	testCtx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()
	correlationClient, err := correlations.NewCorrelationClient(testCtx, &config.WriterConfig{})
	assert.NoError(t, err, "failed to create correlation client")

	hostIdDims := map[string]string{"host": "test", "AWSUniqueId": "randomAWSUniqueId"}
	a := New(5*time.Minute, correlationClient, hostIdDims, "", nil)
	setTime(a, time.Unix(100, 0))

	a.AddSpans(context.Background(), []*trace.Span{
		{
			LocalEndpoint: &trace.Endpoint{
				ServiceName: pointer.String("one"),
			},
			Tags: utils.MergeStringMaps(hostIdDims, map[string]string{"environment": "environment1"}),
		},
		{
			LocalEndpoint: &trace.Endpoint{
				ServiceName: pointer.String("two"),
			},
			Tags: utils.MergeStringMaps(hostIdDims, map[string]string{"environment": "environment2"}),
		},
		{
			LocalEndpoint: &trace.Endpoint{
				ServiceName: pointer.String("three"),
			},
			Tags: utils.MergeStringMaps(hostIdDims, map[string]string{"environment": "environment3"}),
		},
	})

	assert.Equal(t, int64(3), a.hostServiceCache.ActiveCount, "activeServiceCount is not properly tracked")
	assert.Equal(t, int64(3), a.hostEnvironmentCache.ActiveCount, "activeEnvironmentCount is not properly tracked")

	advanceTime(a, 4)

	a.AddSpans(context.Background(), []*trace.Span{
		{
			LocalEndpoint: &trace.Endpoint{
				ServiceName: pointer.String("two"),
			},
			Tags: utils.MergeStringMaps(hostIdDims, map[string]string{"environment": "environment2"}),
		},
	})

	advanceTime(a, 2)
	a.Purge()
	dps := a.CorrelationDatapoints()
	assert.Len(t, dps, 1, "Expected one datapoint")
	assert.Equal(t, dps[0].Dimensions["sf_hasService"], "two", "expected service two to still be active")

	assert.Equal(t, int64(1), a.hostServiceCache.ActiveCount, "activeServiceCount is not properly tracked")
	assert.Equal(t, int64(1), a.hostEnvironmentCache.ActiveCount, "activeEnvironmentCount is not properly tracked")
	assert.Equal(t, int64(2), a.hostServiceCache.PurgedCount, "purgedServiceCount is not properly tracked")
	assert.Equal(t, int64(2), a.hostEnvironmentCache.PurgedCount, "activeEnvironmentCount is not properly tracked")

	advanceTime(a, 4)
	a.Purge()
	assert.Len(t, a.CorrelationDatapoints(), 0, "Expected all datapoints to be expired")
}

type correlationTestClient struct {
	sync.Mutex
	cors []*correlations.Correlation
}

func (c *correlationTestClient) Start() { /*no-op*/ }
func (c *correlationTestClient) AcceptCorrelation(cl *correlations.Correlation) {
	c.Lock()
	defer c.Unlock()
	c.cors = append(c.cors, cl)
}
func (c *correlationTestClient) getCorrelations() []*correlations.Correlation {
	c.Lock()
	defer c.Unlock()
	return c.cors[:]
}
func (c *correlationTestClient) reset() {
	c.Lock()
	defer c.Unlock()
	c.cors = c.cors[0:]
}

var _ correlations.CorrelationClient = &correlationTestClient{}

func TestCorrelationUpdates(t *testing.T) {
	correlationClient := &correlationTestClient{}
	hostIdDims := map[string]string{"host": "test", "AWSUniqueId": "randomAWSUniqueId"}
	containerLevelIDDims := map[string]string{"kubernetes_pod_uid": "testk8sPodUID", "container_id": "testContainerID"}
	a := New(5*time.Minute, correlationClient, hostIdDims, "", nil)
	setTime(a, time.Unix(100, 0))

	a.AddSpans(context.Background(), []*trace.Span{
		{
			LocalEndpoint: &trace.Endpoint{
				ServiceName: pointer.String("one"),
			},
			Tags: utils.MergeStringMaps(hostIdDims, utils.MergeStringMaps(containerLevelIDDims, map[string]string{"environment": "environment1"})),
		},
		{
			LocalEndpoint: &trace.Endpoint{
				ServiceName: pointer.String("two"),
			},
			Tags: utils.MergeStringMaps(hostIdDims, utils.MergeStringMaps(containerLevelIDDims, map[string]string{"environment": "environment2"})),
		},
		{
			LocalEndpoint: &trace.Endpoint{
				ServiceName: pointer.String("three"),
			},
			Tags: utils.MergeStringMaps(hostIdDims, utils.MergeStringMaps(containerLevelIDDims, map[string]string{"environment": "environment3"})),
		},
	})

	assert.Equal(t, int64(3), a.hostServiceCache.ActiveCount, "activeServiceCount is not properly tracked")
	assert.Equal(t, int64(3), a.hostEnvironmentCache.ActiveCount, "activeEnvironmentCount is not properly tracked")

	numEnvironments := 3
	numServices := 3
	numHostIDDimCorrelations := len(hostIdDims) * (numEnvironments + numServices)
	numContainerLevelCorrelations := len(containerLevelIDDims) * (numEnvironments + numServices)
	totalExpectedCorrelations := numHostIDDimCorrelations + numContainerLevelCorrelations
	assert.Equal(t, totalExpectedCorrelations, len(correlationClient.getCorrelations()), "#of correlation requests do not match")

	// TODO @scotts @charlie actually look at the correlations returned and make sure they are what we expect

	advanceTime(a, 6)
	a.Purge()
	dps := a.CorrelationDatapoints()
	assert.Len(t, dps, 0, "Expected all datapoints to be expired")
	assert.Equal(t, int64(0), a.hostServiceCache.ActiveCount, "activeServiceCount is not properly tracked")
	assert.Equal(t, int64(0), a.hostEnvironmentCache.ActiveCount, "activeEnvironmentCount is not properly tracked")
	assert.Equal(t, int64(3), a.hostServiceCache.PurgedCount, "purgedServiceCount is not properly tracked")
	assert.Equal(t, int64(3), a.hostEnvironmentCache.PurgedCount, "activeEnvironmentCount is not properly tracked")
}
