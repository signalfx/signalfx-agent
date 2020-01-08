package service

import (
	"testing"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestRetrievePoints(t *testing.T) {
	gateway := newFakeGateway()
	log := logrus.WithField("monitorType", "vsphere-test")
	inventorySvc := NewInventorySvc(gateway, log)
	metricsSvc := NewMetricsService(gateway, log)
	infoSvc := NewVSphereInfoService(inventorySvc, metricsSvc)
	vsphereInfo, _ := infoSvc.RetrieveVSphereInfo()
	svc := NewPointsSvc(gateway, log)
	pts, _ := svc.RetrievePoints(vsphereInfo, 1)
	pt := pts[0]
	require.Equal(t, "vsphere.cpu_core_utilization_percent", pt.Metric)
	require.Equal(t, datapoint.Count, pt.MetricType)
	require.EqualValues(t, 1.11, pt.Value)
}
