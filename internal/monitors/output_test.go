package monitors

import (
	"testing"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/stretchr/testify/assert"
)

func helperTestMonitorOuput() (*monitorOutput, error) {
	config := &config.MonitorConfig{}
	var metadata *Metadata

	monFiltering, err := newMonitorFiltering(config, metadata)
	if err != nil {
		return nil, err
	}

	output := &monitorOutput{
		monitorType:      "testMonitor",
		monitorID:        "testMonitor1",
		monitorFiltering: monFiltering,
	}
	return output, nil
}

func TestSendDatapoint(t *testing.T) {
	// Setup our 'fixture' super basic monitorOutput
	testMO, err := helperTestMonitorOuput()
	assert.Nil(t, err)

	// And our Datapoint channel to receive Datapoints
	dpChan := make(chan *datapoint.Datapoint)
	testMO.dpChan = dpChan

	// And our reference timestamp
	dpTimestamp := time.Now()

	// Create a test Datapoint
	testDp := datapoint.New("test.metric.name", nil, datapoint.NewIntValue(1), datapoint.Gauge, dpTimestamp)

	// Send the datapoint
	go func() { testMO.SendDatapoint(testDp) }()

	// Receive the datapoint
	resultDp := <-dpChan

	// Make sure it's come through as expected
	assert.Equal(t, "test.metric.name", resultDp.Metric)
	assert.Equal(t, map[string]string{}, resultDp.Dimensions)
	assert.Equal(t, datapoint.NewIntValue(1), resultDp.Value)
	assert.Equal(t, datapoint.Gauge, resultDp.MetricType)
	assert.Equal(t, dpTimestamp, resultDp.Timestamp)

	// Let's add some extra dimensions to our monitorOutput
	testMO.extraDims = map[string]string{"testDim1": "testValue1"}

	// Resend the datapoint
	go func() { testMO.SendDatapoint(testDp) }()

	// Receive the datapoint
	resultDp = <-dpChan

	// Make sure it's come through as expected
	assert.Equal(t, map[string]string{"testDim1": "testValue1"}, resultDp.Dimensions)

	// Add some dimensions in the test Datapoint
	go func() {
		testDp.Dimensions = map[string]string{"testDim2": "testValue2"}
		testMO.SendDatapoint(testDp)
	}()

	// Receive the datapoint
	resultDp = <-dpChan

	// Make sure it's come through as expected
	assert.Equal(t, map[string]string{"testDim1": "testValue1", "testDim2": "testValue2"}, resultDp.Dimensions)

	// Test using the dimension transformation
	testMO.dimensionTransformations = map[string]string{"testDim2": "testDim3"}

	// Send the datapoint with a dimension that matches our transform
	go func() {
		testDp.Dimensions = map[string]string{"testDim2": "testValue2"}
		testMO.SendDatapoint(testDp)
	}()

	// Receive the datapoint
	resultDp = <-dpChan

	// Make sure it's come through as expected
	assert.Equal(t, map[string]string{"testDim1": "testValue1", "testDim3": "testValue2"}, resultDp.Dimensions)

	// Test using the dimension transformation to remove an unwanted dimension
	testMO.dimensionTransformations = map[string]string{"highCardDim": ""}

	// Send the datapoint with a matching dimension
	go func() {
		testDp.Dimensions = map[string]string{"highCardDim": "highCardValue"}
		testMO.SendDatapoint(testDp)
	}()

	// Receive the datapoint
	resultDp = <-dpChan

	// Make sure it's come through as expected
	assert.Equal(t, map[string]string{"testDim1": "testValue1"}, resultDp.Dimensions)
}
