package kubernetes

import (
	"fmt"

	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// ClusterState makes extensive use of the K8s client's "informer" framework,
// which is fairly poorly documented but seems to work pretty well and is well
// suited to our use case.
type ClusterState struct {
	clientset   *k8s.Clientset
	indexers    map[string]cache.Indexer
	controllers map[string]*cache.Controller
	stoppers    map[string]chan struct{}

	ChangeFunc func(runtime.Object, runtime.Object)
}

func newClusterState(clientset *k8s.Clientset) *ClusterState {
	return &ClusterState{
		clientset:   clientset,
		indexers:    make(map[string]cache.Indexer),
		controllers: make(map[string]*cache.Controller),
		stoppers:    make(map[string]chan struct{}),
	}
}

// Stop all running goroutines.
func (cs *ClusterState) Stop() {
	for k, s := range cs.stoppers {
		if s != nil {
			s <- struct{}{}
			cs.stoppers[k] = nil
		}
	}
}

// EnsureAllStarted starts syncing any resource that isn't already being synced
func (cs *ClusterState) EnsureAllStarted() {
	if cs.indexers["pods"] == nil {
		cs.StartSyncing(&v1.Pod{})
	}
	if cs.indexers["daemonsets"] == nil {
		cs.StartSyncing(&v1beta1.DaemonSet{})
	}
	if cs.indexers["deployments"] == nil {
		cs.StartSyncing(&v1beta1.Deployment{})
	}
	if cs.indexers["replicationcontrollers"] == nil {
		cs.StartSyncing(&v1.ReplicationController{})
	}
	if cs.indexers["replicasets"] == nil {
		cs.StartSyncing(&v1beta1.ReplicaSet{})
	}
}

// GetAgentPods returns only running SignalFx agent pods, or error if pods
// haven't been synced yet
func (cs *ClusterState) GetAgentPods() ([]*v1.Pod, error) {
	if cs.indexers["pods"] == nil {
		return nil, fmt.Errorf("Pods have not been synced yet")
	}

	objs, err := cs.indexers["pods"].ByIndex("appLabel", "signalfx-agent")
	if err != nil {
		return nil, err
	}

	pods := make([]*v1.Pod, 0, len(objs))
	for _, p := range objs {
		pods = append(pods, (p.(*v1.Pod)))
	}

	return pods, nil
}

// StartSyncing starts syncing a single resource.  Useful to only sync pods to
// determine if an instance is a reporter or not.
func (cs *ClusterState) StartSyncing(resType runtime.Object) {
	var resName string
	var indexers cache.Indexers = map[string]cache.IndexFunc{}
	var client rest.Interface

	switch resType.(type) {
	case *v1.Pod:
		resName = "pods"
		client = cs.clientset.Core().RESTClient()
		indexers = map[string]cache.IndexFunc{
			"appLabel": func(obj interface{}) ([]string, error) {
				pod := obj.(*v1.Pod)
				// Ignore non-running agents
				if pod.Status.Phase == v1.PodRunning {
					return []string{pod.Labels["app"]}, nil
				}
				return []string{}, nil
			},
		}
	case *v1.ReplicationController:
		resName = "replicationcontrollers"
		client = cs.clientset.Core().RESTClient()
	case *v1beta1.DaemonSet:
		resName = "daemonsets"
		client = cs.clientset.ExtensionsV1beta1().RESTClient()
	case *v1beta1.Deployment:
		resName = "deployments"
		client = cs.clientset.ExtensionsV1beta1().RESTClient()
	case *v1beta1.ReplicaSet:
		resName = "replicasets"
		client = cs.clientset.ExtensionsV1beta1().RESTClient()
	}

	// Stop previous informers
	if cs.stoppers[resName] != nil {
		cs.stoppers[resName] <- struct{}{}
	} else {
		cs.stoppers[resName] = make(chan struct{})
	}

	watchList := cache.NewListWatchFromClient(client, resName, api.NamespaceAll, fields.Everything())

	cs.indexers[resName], cs.controllers[resName] = cache.NewIndexerInformer(
		watchList,
		resType,
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				cs.ChangeFunc(nil, obj.(runtime.Object))
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				cs.ChangeFunc(oldObj.(runtime.Object), newObj.(runtime.Object))
			},
			DeleteFunc: func(obj interface{}) {
				cs.ChangeFunc(obj.(runtime.Object), nil)
			},
		},
		indexers)

	go cs.controllers[resName].Run(cs.stoppers[resName])
}
