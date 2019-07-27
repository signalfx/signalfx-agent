package monitors

import (
	"testing"
	"time"

	"github.com/signalfx/golib/datapoint"
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

	// Create a test Datapoint
	testDp := datapoint.New("test.metric.name", nil, datapoint.NewIntValue(1), datapoint.Gauge, time.Now())

	// Send the datapoint
	go func() { testMO.SendDatapoint(testDp) }()

	// Receive the datapoint
	resultDp := <-dpChan

	// Make sure it's come through unscathed
	assert.Equal(t, testDp, resultDp)

	// Let's add some extra dimensions to our monitorOutput
	testMO.extraDims = map[string]string{"testDim1": "testValue1"}

	// Resend the datapoint
	go func() { testMO.SendDatapoint(testDp) }()

	// Receive the datapoint
	resultDp = <-dpChan

	// Make sure it's come through unscathed
	assert.Equal(t, map[string]string{"testDim1": "testValue1"}, resultDp.Dimensions)

	// Now let's add some dimensions in the test Datapoint
	go func() {
		testDp.Dimensions = map[string]string{"testDim2": "testValue2"}
		testMO.SendDatapoint(testDp)
	}()

	// Receive the datapoint
	resultDp = <-dpChan

	// Make sure it's come through unscathed
	assert.Equal(t, map[string]string{"testDim1": "testValue1", "testDim2": "testValue2"}, resultDp.Dimensions)

	// Let's now test removing an unwanted dimension
	testMO.excludeDims = []string{"highCardDim"}

	// Send the datapoint with a high cardinality dimension
	go func() {
		testDp.Dimensions = map[string]string{"highCardDim": "highCardValue"}
		testMO.SendDatapoint(testDp)
	}()

	// Receive the datapoint
	resultDp = <-dpChan

	// Make sure it's come through unscathed
	assert.Equal(t, map[string]string{"testDim1": "testValue1"}, resultDp.Dimensions)

}
