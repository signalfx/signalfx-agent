package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPopulateInvMetrics(t *testing.T) {
	gateway := newFakeGateway()
	log := getTestingLog()
	metricsSvc := NewMetricsService(gateway, log)
	inventorySvc := NewInventorySvc(gateway, log)
	inv, _ := inventorySvc.RetrieveInventory()
	metricsSvc.PopulateInvMetrics(inv)
	invObj := inv.Objects[0]
	perfMetricID := invObj.MetricIds[0]
	require.EqualValues(t, "instance-0", perfMetricID.Instance)
}

func TestRetrievePerfCounterIndex(t *testing.T) {
	gateway := newFakeGateway()
	metricsSvc := NewMetricsService(gateway, getTestingLog())
	idx, _ := metricsSvc.RetrievePerfCounterIndex()
	metric := idx[42]
	require.Equal(t, "vsphere.cpu_core_utilization_percent", metric.MetricName)
}

func TestDotsToUnderscores(t *testing.T) {
	replaced := dotsToUnderscores("aaa.bbb.ccc")
	require.Equal(t, "aaa_bbb_ccc", replaced)
}
