package vsphere

import (
	"context"

	"github.com/signalfx/signalfx-agent/pkg/monitors/vsphere/model"
	"github.com/sirupsen/logrus"
)

type runner struct {
	ctx                   context.Context
	log                   *logrus.Entry
	monitor               *Monitor
	conf                  *model.Config
	vsm                   *vSphereMonitor
	vsphereReloadInterval int //seconds
}

func newRunner(ctx context.Context, log *logrus.Entry, conf *model.Config, monitor *Monitor) runner {
	vsphereReloadInterval := int(conf.InventoryRefreshInterval.Seconds())
	vsm := newVsphereMonitor(log)
	return runner{
		ctx:                   ctx,
		log:                   log,
		monitor:               monitor,
		conf:                  conf,
		vsphereReloadInterval: vsphereReloadInterval,
		vsm:                   vsm,
	}
}

// Called periodically. This is the entry point to the vSphere monitor.
func (r *runner) run() {
	err := r.vsm.firstTimeSetup(r.ctx, r.conf)
	if err != nil {
		r.log.WithError(err).Error("firstTimeSetup failed")
		return
	}
	dps := r.vsm.retrieveDatapoints()
	r.monitor.Output.SendDatapoints(dps...)
	if r.vsm.isTimeForVSphereInfoReload(r.vsphereReloadInterval) {
		r.vsm.reloadVSphereInfo()
	}
}
