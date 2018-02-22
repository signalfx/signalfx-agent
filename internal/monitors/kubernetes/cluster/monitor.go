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
// 1) With the default configuration, this plugin will watch the current list
// of our agent pods, and if and only if it is the first pod in the list,
// sorted alphabetically by pod name ascending, it will be a reporter. Each
// instance of the agent will check upon each reporting interval whether it is
// the first such pod and begin reporting if it finds that it has become the
// reporter.  This method requires one long-running connection to the K8s API
// server per node (assuming the agent is running on all nodes).
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

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

// Config for the K8s monitor
type Config struct {
	config.MonitorConfig
	AlwaysClusterReporter bool                  `yaml:"alwaysClusterReporter"`
	KubernetesAPI         *kubernetes.APIConfig `yaml:"kubernetesAPI" default:"{}"`
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
