package vsphere

import (
	"context"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/monitors/vsphere/model"
	"github.com/signalfx/signalfx-agent/pkg/monitors/vsphere/service"
	"github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
)

// Encapsulates services and the current state of the vSphere monitor.
type vSphereMonitor struct {
	conf *model.Config
	log  *logrus.Entry

	invSvc    *service.InventorySvc
	metricSvc *service.MetricsSvc
	vsInfoSvc *service.VSphereInfoService
	timeSvc   *service.TimeSvc
	ptsSvc    *service.PointsSvc

	vSphereInfo            *model.VsphereInfo
	lastVsphereLoadTime    time.Time
	lastPointRetrievalTime time.Time
}

func newVsphereMonitor(conf *model.Config, log *logrus.Entry) *vSphereMonitor {
	return &vSphereMonitor{
		conf: conf,
		log:  log,
	}
}

// Logs into vSphere, wires up service objects, and retrieves vSphereInfo (inventory, available metrics, and metric index).
func (vsm *vSphereMonitor) firstTimeSetup(ctx context.Context) error {
	if !vsm.lastVsphereLoadTime.IsZero() {
		return nil
	}
	authSvc := service.NewAuthService(vsm.log)
	client, err := authSvc.LogIn(ctx, vsm.conf)
	if err != nil {
		return err
	}

	vsm.wireUpServices(ctx, client)

	vsm.vSphereInfo, err = vsm.vsInfoSvc.RetrieveVSphereInfo()
	if err != nil {
		return err
	}
	currentTime, err := vsm.timeSvc.RetrieveCurrentTime()
	if err != nil {
		return err
	}
	vsm.lastVsphereLoadTime = *currentTime
	return nil
}

// Creates the service objects and assigns them to the vSphereMonitor struct.
func (vsm *vSphereMonitor) wireUpServices(ctx context.Context, client *govmomi.Client) {
	gateway := service.NewGateway(ctx, client, vsm.log)
	vsm.ptsSvc = service.NewPointsSvc(gateway, vsm.log, vsm.conf.PerfBatchSize)
	vsm.invSvc = service.NewInventorySvc(gateway, vsm.log)
	vsm.metricSvc = service.NewMetricsService(gateway, vsm.log)
	vsm.timeSvc = service.NewTimeSvc(gateway)
	vsm.vsInfoSvc = service.NewVSphereInfoService(vsm.invSvc, vsm.metricSvc)
}

// Retrieves datapoints for all the inventory for the number of 20-second intervals available since the last datapoint
// retrieval.
func (vsm *vSphereMonitor) retrieveDatapoints() []*datapoint.Datapoint {
	numSamples, err := vsm.getNumSamplesReqd()
	if err != nil {
		vsm.log.WithError(err).Error("Failed to load getNumSamplesReqd")
		return nil
	}
	if numSamples == 0 {
		return nil
	}
	dps, latestRetrievalTime := vsm.ptsSvc.RetrievePoints(vsm.vSphereInfo, numSamples)
	if !latestRetrievalTime.IsZero() {
		vsm.lastPointRetrievalTime = latestRetrievalTime
	}
	return dps
}

// Traverses the vSphere inventory and saves the result in vSphereInfo (hosts, VMs, available metrics, and metric index).
func (vsm *vSphereMonitor) reloadVSphereInfo() {
	var err error
	vsm.vSphereInfo, err = vsm.vsInfoSvc.RetrieveVSphereInfo()
	if err != nil {
		vsm.log.WithError(err).Error("Failed to load vSphereInfo")
		return
	}
	currentTime, err := vsm.timeSvc.RetrieveCurrentTime()
	if err != nil {
		vsm.log.WithError(err).Error("Failed to retrieve current time")
		return
	}
	vsm.lastVsphereLoadTime = *currentTime
}

// Compares the last vSphereInfo load time to the vSphere info reload interval, returning whether more time has elapsed
// than the configured duration.
func (vsm *vSphereMonitor) isTimeForVSphereInfoReload(vsphereReloadIntervalSeconds int) bool {
	secondsSinceLastVsReload := int(time.Since(vsm.lastVsphereLoadTime).Seconds())
	timeForReload := secondsSinceLastVsReload > vsphereReloadIntervalSeconds
	vsm.log.WithFields(logrus.Fields{
		"secondsSinceLastVsReload": secondsSinceLastVsReload,
		"vsphereReloadInterval":    vsphereReloadIntervalSeconds,
	}).Debugf("Time for vs reload = %t", timeForReload)
	return timeForReload
}

// Returns the number of 20-second intervals available in vSphere since the last time points were retrieved
func (vsm *vSphereMonitor) getNumSamplesReqd() (int32, error) {
	if vsm.lastPointRetrievalTime.IsZero() {
		return 1, nil
	}
	currentTime, err := vsm.timeSvc.RetrieveCurrentTime()
	if err != nil {
		return 0, err
	}
	fSecondsSinceLastInterval := currentTime.Sub(vsm.lastPointRetrievalTime).Seconds()
	intSecondsSinceLastInterval := int32(fSecondsSinceLastInterval)
	numSamples := intSecondsSinceLastInterval / model.RealtimeMetricsInterval
	vsm.log.WithFields(logrus.Fields{
		"currentTime":                 currentTime,
		"lastTime":                    vsm.lastPointRetrievalTime,
		"fSecondsSinceLastInterval":   fSecondsSinceLastInterval,
		"intSecondsSinceLastInterval": intSecondsSinceLastInterval,
	}).Debugf("numSamples = %d", numSamples)
	return numSamples, nil
}
