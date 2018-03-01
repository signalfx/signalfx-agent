package cluster

import (
	log "github.com/sirupsen/logrus"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// State makes extensive use of the K8s client's "informer" framework,
// which is fairly poorly documented but seems to work pretty well and is well
// suited to our use case.
type State struct {
	clientset   *k8s.Clientset
	indexers    map[string]cache.Indexer
	controllers map[string]cache.Controller
	stoppers    map[string]chan struct{}

	ChangeFunc func(runtime.Object, runtime.Object)
}

func newState(clientset *k8s.Clientset) *State {
	return &State{
		clientset:   clientset,
		controllers: make(map[string]cache.Controller),
		stoppers:    make(map[string]chan struct{}),
	}
}

// Start starts syncing any resource that isn't already being synced
func (cs *State) Start() {
	log.Info("Starting K8s API resource sync")

	coreClient := cs.clientset.CoreV1().RESTClient()
	extV1beta1Client := cs.clientset.ExtensionsV1beta1().RESTClient()

	cs.beginSyncForType(&v1.Pod{}, "pods", coreClient)
	cs.beginSyncForType(&v1beta1.DaemonSet{}, "daemonsets", extV1beta1Client)
	cs.beginSyncForType(&v1beta1.Deployment{}, "deployments", extV1beta1Client)
	cs.beginSyncForType(&v1.ReplicationController{}, "replicationcontrollers", coreClient)
	cs.beginSyncForType(&v1beta1.ReplicaSet{}, "replicasets", extV1beta1Client)
	cs.beginSyncForType(&v1.Node{}, "nodes", coreClient)
	cs.beginSyncForType(&v1.Namespace{}, "namespaces", coreClient)
}

func (cs *State) beginSyncForType(resType runtime.Object, resName string, client rest.Interface) {
	cs.stoppers[resName] = make(chan struct{})

	watchList := cache.NewListWatchFromClient(client, resName, v1.NamespaceAll, fields.Everything())

	_, cs.controllers[resName] = cache.NewInformer(
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
		})

	go cs.controllers[resName].Run(cs.stoppers[resName])
}

// Stop all running goroutines. There is a bug/limitation in the k8s go
// client's Controller where goroutines are leaked even when using the stop
// channel properly.
// See https://github.com/kubernetes/client-go/blob/release-6.0/tools/cache/controller.go#L144
func (cs *State) Stop() {
	log.Info("Stopping all K8s API resource sync")
	for k := range cs.stoppers {
		close(cs.stoppers[k])
		delete(cs.stoppers, k)
		delete(cs.controllers, k)
	}
}
