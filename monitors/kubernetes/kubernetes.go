package kubernetes

import (
	"log"
	"sort"
	"time"

	"golang.org/x/net/context"

	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/neo-agent/core/filters"
)

// Kubernetes is distinct from the plugin type for less coupling to
// neo-agent
type Kubernetes struct {
	sfxClient *SFXClient
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
	stop   chan struct{}
	Filter *filters.FilterSet
}

// NewKubernetes creates a new monitor instance
func NewKubernetes(k8sClient *k8s.Clientset,
	sfxClient *SFXClient,
	interval uint,
	alwaysClusterReporter bool,
	thisPodName string) *Kubernetes {
	datapointCache := newDatapointCache()

	clusterState := newClusterState(k8sClient)
	clusterState.ChangeFunc = datapointCache.HandleChange

	return &Kubernetes{
		sfxClient:             sfxClient,
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

func (km *Kubernetes) filterDatapoints(dps []*datapoint.Datapoint) []*datapoint.Datapoint {
	newDps := make([]*datapoint.Datapoint, 0, len(dps))
	for _, dp := range dps {
		if km.Filter != nil && !km.Filter.Matches(dp, monitorType) {
			newDps = append(newDps, dp)
		}
	}
	return newDps
}

// Synchonously send all of the cached datapoints to ingest
func (km *Kubernetes) sendLatestDatapoints() {
	km.datapointCache.Mutex.Lock()
	defer km.datapointCache.Mutex.Unlock()

	dps := updateTimestamps(km.filterDatapoints(km.datapointCache.AllDatapoints()))

	// This sends synchonously despite what the first param might seem to
	// indicate
	err := km.sfxClient.AddDatapoints(context.Background(), dps)
	if err != nil {
		log.Print("Error shipping datapoints to SignalFx: ", err)
		// If there is an error sending datapoints then just forget about them.
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
			log.Print("Unexpected error getting agent pods: ", err)
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
