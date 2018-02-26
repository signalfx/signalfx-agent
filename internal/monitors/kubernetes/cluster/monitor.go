// Package cluster contains a Kubernetes cluster monitor.
//
// This plugin collects high level metrics about a K8s cluster and sends them
// to SignalFx.  The basic technique is to pull data from the K8s API and keep
// up-to-date copies of datapoints for each metric that we collect and then
// ship them off at the end of each reporting interval.  The K8s streaming
// watch API is used to effeciently maintain the state between read intervals
// (see `clusterstate.go`).
//
// This plugin should only be run at one place in the cluster, or else metrics
// would be duplicated.  This plugin supports two ways of ensuring that:
//
// 2) You can simply pass a config flag `alwaysClusterReporter` with value of
// `true` to this plugin and it will always report cluster metrics.  This
// method uses less cluster resources (e.g. network sockets, watches on the api
// server) but requires special case configuration for a single agent in the
// cluster, which may be more error prone.
//
// This plugin requires read-only access to the K8s API.
package cluster

import (
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime"
	k8s "k8s.io/client-go/kubernetes"

	"github.com/signalfx/signalfx-agent/internal/core/common/dpmeta"
	"github.com/signalfx/signalfx-agent/internal/core/common/kubernetes"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/kubernetes/cluster/metrics"
	"github.com/signalfx/signalfx-agent/internal/monitors/kubernetes/leadership"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
)

const (
	monitorType = "kubernetes-cluster"
)

// MONITOR(kubernetes-cluster): Collects cluster-level metrics from the
// Kubernetes API server.  It uses the _watch_ functionality of the K8s API
// to listen for updates about the cluster and maintains a cache of metrics
// that get sent on a regular interval.
//
// Since the agent is generally running in multiple places in a K8s cluster and
// since it is generally more convenient to share the same configuration across
// all agent instances, this monitor by default makes use of a leader election
// process to ensure that it is the only agent sending metrics in a cluster.
// All of the agents running in the same namespace that have this monitor
// configured will decide amongst themselves which should send metrics for this
// monitor, and the rest will stand by ready to activate if the leader agent
// dies.  You can override leader election by setting the config option
// `alwaysClusterReporter` to true, which will make the monitor always report
// metrics.
//
// This monitor is similar to
// [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics), and
// sends many of the same metrics, but in a way that is less verbose and better
// fitted for the SignalFx backend.

// DIMENSION(kubernetes_namespace): The namespace of the resource that the metric
// describes

// DIMENSION(kubernetes_pod_uid): The UID of the pod that the metric describes

// DIMENSION(metric_source): This is always set to `kubernetes`

// DIMENSION(kubernetes_name): The name of the resource that the metric
// describes

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

// Config for the K8s monitor
type Config struct {
	config.MonitorConfig
	// If `true`, leader election is skipped and metrics are always reported.
	AlwaysClusterReporter bool `yaml:"alwaysClusterReporter"`
	// Config for the K8s API client
	KubernetesAPI *kubernetes.APIConfig `yaml:"kubernetesAPI" default:"{}"`
}

// Validate the k8s-specific config
func (c *Config) Validate() error {
	return c.KubernetesAPI.Validate()
}

// Monitor for K8s Cluster Metrics.  Also handles syncing certain properties
// about pods.
type Monitor struct {
	config      *Config
	Output      types.Output
	thisPodName string
	// Since most datapoints will stay the same or only slightly different
	// across reporting intervals, reuse them
	datapointCache *metrics.DatapointCache
	k8sClient      *k8s.Clientset
	stop           chan struct{}
}

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Configure is called by the plugin framework when configuration changes
func (m *Monitor) Configure(config *Config) error {
	m.config = config

	k8sClient, err := kubernetes.MakeClient(config.KubernetesAPI)
	if err != nil {
		return errors.Wrapf(err, "Could not create K8s API client")
	}

	m.k8sClient = k8sClient
	m.datapointCache = metrics.NewDatapointCache()
	m.stop = make(chan struct{})

	m.Start()

	return nil
}

// Start starts syncing resources and sending datapoints to ingest
func (m *Monitor) Start() error {
	ticker := time.NewTicker(time.Second * time.Duration(m.config.IntervalSeconds))

	shouldReport := m.config.AlwaysClusterReporter

	clusterState := newState(m.k8sClient)
	clusterState.ChangeFunc = func(oldObj, newObj runtime.Object) {
		m.datapointCache.HandleChange(oldObj, newObj)
	}

	var leaderCh <-chan bool
	var unregister func()

	if m.config.AlwaysClusterReporter {
		log.Error("STARTING")
		clusterState.Start()
	} else {
		var err error
		leaderCh, unregister, err = leadership.RequestLeaderNotification(m.k8sClient.CoreV1())
		if err != nil {
			return err
		}
	}

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-m.stop:
				if unregister != nil {
					unregister()
				}
				clusterState.Stop()
				return
			case isLeader := <-leaderCh:
				if isLeader {
					shouldReport = true
					clusterState.Start()
				} else {
					shouldReport = false
					clusterState.Stop()
				}
			case <-ticker.C:
				if shouldReport {
					m.sendLatestDatapoints()
					m.sendLatestProps()
				}
			}
		}
	}()

	return nil
}

// Synchonously send all of the cached datapoints to ingest
func (m *Monitor) sendLatestDatapoints() {
	dps := m.datapointCache.AllDatapoints()

	now := time.Now()
	for i := range dps {
		dps[i].Timestamp = now
		dps[i].Meta[dpmeta.NotHostSpecificMeta] = true
		m.Output.SendDatapoint(dps[i])
	}
}

func (m *Monitor) sendLatestProps() {
	dimProps := m.datapointCache.AllDimProperties()

	for i := range dimProps {
		m.Output.SendDimensionProps(dimProps[i])
	}
}

// Shutdown halts everything that is syncing
func (m *Monitor) Shutdown() {
	if m.stop != nil {
		close(m.stop)
	}
}
