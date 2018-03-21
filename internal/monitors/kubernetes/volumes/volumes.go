package volumes

import (
	"fmt"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/signalfx-agent/internal/core/common/kubelet"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
	stats "k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
)

const monitorType = "kubernetes-volumes"

// MONITOR(kubernetes-volumes): This monitor sends usage stats about volumes
// mounted to Kubernetes pods (e.g. free space/inodes).  This information is
// gotten from the Kubelet /stats/summary endpoint.  The normal `collectd/df`
// monitor generally will not report Persistent Volume usage metrics because
// those volumes are not seen by the agent since they can be mounted
// dynamically and older versions of K8s don't support mount propagation of
// those mounts to the agent container.

// DIMENSION(volume): The volume name as given in the pod spec under `volumes`
// DIMENSION(kubernetes_pod_uid): The UID of the pod that has this volume
// DIMENSION(kubernetes_pod_name): The name of the pod that has this volume
// DIMENSION(kubernetes_namespace): The namespace of the pod that has this volume

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

type Config struct {
	config.MonitorConfig
	// Kubelet client configuration
	KubeletAPI kubelet.APIConfig `yaml:"kubeletAPI" default:""`
}

// Monitor for K8s Cluster Metrics.  Also handles syncing certain properties
// about pods.
type Monitor struct {
	Output types.Output
	stop   func()
	client *kubelet.Client
}

// Configure the monitor and kick off event syncing
func (m *Monitor) Configure(conf *Config) error {
	var err error
	m.client, err = kubelet.NewClient(&conf.KubeletAPI)
	if err != nil {
		return err
	}

	m.stop = utils.RunOnInterval(func() {
		dps, err := m.getVolumeMetrics()
		if err != nil {
			logger.WithError(err).Error("Could not get volume metrics")
			return
		}

		for i := range dps {
			m.Output.SendDatapoint(dps[i])
		}
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

func (m *Monitor) getVolumeMetrics() ([]*datapoint.Datapoint, error) {
	summary, err := m.getSummary()
	if err != nil {
		return nil, err
	}

	var dps []*datapoint.Datapoint
	for _, p := range summary.Pods {
		for _, v := range p.VolumeStats {
			dims := map[string]string{
				"volume":               v.Name,
				"kubernetes_pod_uid":   p.PodRef.UID,
				"kubernetes_pod_name":  p.PodRef.Name,
				"kubernetes_namespace": p.PodRef.Namespace,
			}
			if v.AvailableBytes != nil {
				// uint64 -> int64 conversion should be safe since disk sizes
				// aren't going to get that big for a long time.
				dps = append(dps, sfxclient.Gauge("kubernetes.volume_available_bytes", dims, int64(*v.AvailableBytes)))
			}
			if v.CapacityBytes != nil {
				dps = append(dps, sfxclient.Gauge("kubernetes.volume_capacity_bytes", dims, int64(*v.CapacityBytes)))
			}
		}
	}
	return dps, nil
}

func (m *Monitor) getSummary() (*stats.Summary, error) {
	req, err := m.client.NewRequest("POST", "/stats/summary/", nil)
	if err != nil {
		return nil, err
	}

	var summary stats.Summary
	err = m.client.DoRequestAndSetValue(req, &summary)
	if err != nil {
		return nil, fmt.Errorf("failed to get summary stats from Kubelet URL %q: %v", req.URL.String(), err)
	}

	return &summary, nil
}

func (m *Monitor) Shutdown() {
	if m.stop != nil {
		m.stop()
	}
}
