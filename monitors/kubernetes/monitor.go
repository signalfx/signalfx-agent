// Package kubernetes contains a Kubernetes monitor.
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
package kubernetes

import (
	"os"
	"sort"
	"time"

	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/runtime"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/neo-agent/core/common/kubernetes"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/writer"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/kubernetes/metrics"
	"github.com/signalfx/neo-agent/utils"

	"sync"
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
func (c *Config) Validate() bool {
	if !c.KubernetesAPI.Validate() {
		return false
	}
	return true
}

// Monitor for K8s Cluster Metrics.  Also handles syncing certain properties
// about pods.
type Monitor struct {
	config      *Config
	lock        sync.Mutex
	DPs         chan<- *datapoint.Datapoint
	DimProps    chan<- *writer.DimProperties
	thisPodName string
	// Since most datapoints will stay the same or only slightly different
	// across reporting intervals, reuse them
	datapointCache *metrics.DatapointCache
	clusterState   *ClusterState
	stop           chan struct{}
}

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Configure is called by the plugin framework when configuration changes
func (m *Monitor) Configure(config *Config) bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.config = config

	m.Shutdown()

	k8sClient, err := kubernetes.MakeClient(config.KubernetesAPI)
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err,
		}).Error("Could not create K8s API client")
		return false
	}

	// We need to know the pod name if we aren't always reporting
	if !config.AlwaysClusterReporter {
		var ok bool
		m.thisPodName, ok = os.LookupEnv("MY_POD_NAME")
		if !ok {
			logger.Error("This pod's name is not known! Please inject the envvar MY_POD_NAME " +
				"via a config fieldRef in your K8s agent resource config")
			return false
		}
	}

	m.datapointCache = metrics.NewDatapointCache()

	m.clusterState = newClusterState(k8sClient)
	m.clusterState.ChangeFunc = func(oldObj, newObj runtime.Object) {
		m.datapointCache.HandleChange(oldObj, newObj)
		m.syncResourceProperties(newObj)
	}

	m.Start(config.IntervalSeconds)

	return true
}

// Start starts syncing resources and sending datapoints to ingest
func (m *Monitor) Start(intervalSeconds int) error {
	m.clusterState.StartSyncing(&v1.Pod{})

	ticker := time.NewTicker(time.Second * time.Duration(intervalSeconds))

	go func() {
		defer ticker.Stop()
		m.stop = make(chan struct{})

		for {
			select {
			case <-m.stop:
				return
			case <-ticker.C:
				if m.isReporter() {
					log.Debugf("This agent is a K8s cluster reporter")
					m.clusterState.EnsureAllStarted()
					m.sendLatestDatapoints()
				}
			}
		}
	}()

	return nil
}

// Timestamps are updated in place
func updateTimestamps(dps []*datapoint.Datapoint) []*datapoint.Datapoint {
	// Update timestamp
	now := time.Now()
	for _, dp := range dps {
		dp.Timestamp = now
	}

	return dps
}

// Synchonously send all of the cached datapoints to ingest
func (m *Monitor) sendLatestDatapoints() {
	m.datapointCache.Mutex.Lock()
	defer m.datapointCache.Mutex.Unlock()

	dps := updateTimestamps(m.datapointCache.AllDatapoints())

	for i := range dps {
		m.DPs <- dps[i]
	}
}

// We only need one agent to report high-level K8s metrics so we need to
// deterministically choose one without the agents being able to talk to one
// another (for simplified setup).  About the simplest way to do that is to
// have it be the agent with the pod name that is first when all of the names
// are sorted ascending.
func (m *Monitor) isReporter() bool {
	if m.config.AlwaysClusterReporter {
		return true
	}

	agentPods, err := m.clusterState.GetAgentPods()
	if err != nil {
		log.WithError(err).Error("Unexpected error getting agent pods")
		return false
	}

	// This shouldn't really happen, but don't blow up if it does
	if len(agentPods) == 0 {
		return false
	}

	sort.Slice(agentPods, func(i, j int) bool {
		return agentPods[i].Name < agentPods[j].Name
	})

	return agentPods[0].Name == m.thisPodName
}

// SyncResource will accept a resource and set any properties on dimensions
// that do not already have them (at least as far as the agent knows from it's
// cache, it does not actually query the SignalFx API to discover this).
func (m *Monitor) syncResourceProperties(obj runtime.Object) {
	switch res := obj.(type) {
	case *v1.Pod:
		for _, or := range res.OwnerReferences {
			m.DimProps <- &writer.DimProperties{
				Dimension: writer.Dimension{
					Name:  "kubernetes_pod_uid",
					Value: string(res.UID),
				},
				Properties: map[string]string{
					utils.LowercaseFirstChar(or.Kind): or.Name,
				},
			}
		}
	}
}

// Shutdown halts everything that is syncing
func (m *Monitor) Shutdown() {
	if m.stop != nil {
		m.stop <- struct{}{}
	}
	if m.clusterState != nil {
		m.clusterState.Stop()
	}
}
