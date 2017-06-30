package kubernetes

import (
	"golang.org/x/net/context"
	"log"
	"sort"
	"time"

	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
)

// Make a distinction between the plugin type and the acutal monitor for easier
// testing
type KubernetesMonitor struct {
	sfxClient *sfxclient.HTTPSink
	// How often to report metrics to SignalFx
	intervalSeconds uint
	// Since most datapoints will stay the same or only slightly different
	// across reporting intervals, reuse them
	datapointCache *DatapointCache
	clusterState   *ClusterState
	// If true will definitely report K8s metrics, if false will fall back to
	// checking pod name
	alwaysReport bool
	// If running inside K8s, the name of the current pod, otherwise empty string
	thisPodName string
	// Used to stop the main loop
	stop                chan struct{}
	MetricsToExclude    ExclusionSet
	NamespacesToExclude ExclusionSet
}

func NewKubernetesMonitor(k8sClient *k8s.Clientset,
	sfxClient *sfxclient.HTTPSink,
	interval uint,
	alwaysReport bool,
	thisPodName string) *KubernetesMonitor {
	datapointCache := NewDatapointCache()

	clusterState := NewClusterState(k8sClient)
	clusterState.ChangeFunc = datapointCache.HandleChange

	return &KubernetesMonitor{
		sfxClient:       sfxClient,
		datapointCache:  datapointCache,
		clusterState:    clusterState,
		intervalSeconds: interval,
		alwaysReport:    alwaysReport,
		thisPodName:     thisPodName,
		stop:            make(chan struct{}),
	}
}

func (km *KubernetesMonitor) Start() error {
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
func UpdateTimestamps(dps []*datapoint.Datapoint) []*datapoint.Datapoint {
	// Update timestamp
	now := time.Now()
	for _, dp := range dps {
		dp.Timestamp = now
	}

	return dps
}

func (km *KubernetesMonitor) filterDatapoints(dps []*datapoint.Datapoint) []*datapoint.Datapoint {
	newDps := make([]*datapoint.Datapoint, 0, len(dps))
	for _, dp := range dps {
		metricNamePasses := !km.MetricsToExclude.IsExcluded(dp.Metric)
		ns, nsGiven := dp.Dimensions["kubernetes_pod_namespace"]
		// If namespace isn't defined just let it through
		namespacePasses := !nsGiven || !km.NamespacesToExclude.IsExcluded(ns)

		if metricNamePasses && namespacePasses {
			newDps = append(newDps, dp)
		}
	}
	return newDps
}

// Synchonously send all of the cached datapoints to ingest
func (km *KubernetesMonitor) sendLatestDatapoints() {
	km.datapointCache.Mutex.Lock()
	defer km.datapointCache.Mutex.Unlock()

	dps := UpdateTimestamps(km.filterDatapoints(km.datapointCache.AllDatapoints()))
	log.Printf("Pushing %d metrics to SignalFx", len(dps))

	// This sends synchonously despite what the first param might seem to
	// indicate
	err := km.sfxClient.AddDatapoints(context.Background(), dps)
	if err != nil {
		log.Print("Error shipping datapoints to SignalFx: ", err)
		// If there is an error sending datapoints then just forget about them.
	}
}

func (km *KubernetesMonitor) Stop() {
	km.stop <- struct{}{}
	km.clusterState.Stop()
}

// We only need one agent to report high-level K8s metrics so we need to
// deterministically choose one without the agents being able to talk to one
// another (for simplified setup).  About the simplest way to do that is to
// have it be the agent with the pod name that is first when all of the names
// are sorted ascending.
func (km *KubernetesMonitor) isReporter() bool {
	if km.alwaysReport {
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
