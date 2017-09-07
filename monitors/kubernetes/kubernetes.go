package kubernetes

import (
	"sort"
	"time"

	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"

	"github.com/signalfx/golib/datapoint"
	log "github.com/sirupsen/logrus"
)

// Kubernetes is distinct from the plugin type for less coupling to
// neo-agent
type Kubernetes struct {
	dpChan chan<- *datapoint.Datapoint
	// How often to report metrics to SignalFx
	intervalSeconds uint
	// Since most datapoints will stay the same or only slightly different
	// across reporting intervals, reuse them
	datapointCache *DatapointCache
	clusterState   *ClusterState
	// If true will definitely report K8s metrics, if false will fall back to
	// checking pod name
	alwaysClusterReporter bool
	// If running inside K8s, the name of the current pod, otherwise empty string
	thisPodName string
	// Used to stop the main loop
	stop chan struct{}
}

// NewKubernetes creates a new monitor instance
func NewKubernetes(k8sClient *k8s.Clientset,
	dpChan chan<- *datapoint.Datapoint,
	interval uint,
	alwaysClusterReporter bool,
	thisPodName string) *Kubernetes {
	datapointCache := newDatapointCache()

	clusterState := newClusterState(k8sClient)
	clusterState.ChangeFunc = datapointCache.HandleChange

	return &Kubernetes{
		dpChan:                dpChan,
		datapointCache:        datapointCache,
		clusterState:          clusterState,
		intervalSeconds:       interval,
		alwaysClusterReporter: alwaysClusterReporter,
		thisPodName:           thisPodName,
		stop:                  make(chan struct{}),
	}
}

// Start starts syncing resources and sending datapoints to ingest
func (km *Kubernetes) Start() error {
	km.clusterState.StartSyncing(&v1.Pod{})

	ticker := time.NewTicker(time.Second * time.Duration(km.intervalSeconds))

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-km.stop:
				return
			case <-ticker.C:
				if km.isReporter() {
					km.clusterState.EnsureAllStarted()
					km.sendLatestDatapoints()
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
func (km *Kubernetes) sendLatestDatapoints() {
	km.datapointCache.Mutex.Lock()
	defer km.datapointCache.Mutex.Unlock()

	dps := updateTimestamps(km.datapointCache.AllDatapoints())

	for i := range dps {
		km.dpChan <- dps[i]
	}
}

// Stop halts all syncing and sending of metrics to ingest
func (km *Kubernetes) Stop() {
	km.stop <- struct{}{}
	km.clusterState.Stop()
}

// We only need one agent to report high-level K8s metrics so we need to
// deterministically choose one without the agents being able to talk to one
// another (for simplified setup).  About the simplest way to do that is to
// have it be the agent with the pod name that is first when all of the names
// are sorted ascending.
func (km *Kubernetes) isReporter() bool {
	if km.alwaysClusterReporter {
		return true
	} else if km.thisPodName != "" {
		agentPods, err := km.clusterState.GetAgentPods()
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
		return agentPods[0].Name == km.thisPodName
	}

	return false
}
