package cluster

import (
	"context"

	"github.com/signalfx/signalfx-agent/internal/monitors/kubernetes/cluster/metrics"
	"github.com/signalfx/signalfx-agent/internal/utils/k8sutil"
	log "github.com/sirupsen/logrus"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// State makes use of the K8s client's "reflector" helper to watch the API
// server for changes and keep the datapoint cache up to date,
type State struct {
	clientset  *k8s.Clientset
	reflectors map[string]*cache.Reflector
	cancel     func()

	metricCache *metrics.DatapointCache
}

func newState(clientset *k8s.Clientset, metricCache *metrics.DatapointCache) *State {
	return &State{
		clientset:   clientset,
		reflectors:  make(map[string]*cache.Reflector),
		metricCache: metricCache,
	}
}

// Start starts syncing any resource that isn't already being synced
func (cs *State) Start() {
	log.Info("Starting K8s API resource sync")

	var ctx context.Context
	ctx, cs.cancel = context.WithCancel(context.Background())

	coreClient := cs.clientset.CoreV1().RESTClient()
	extV1beta1Client := cs.clientset.ExtensionsV1beta1().RESTClient()

	cs.beginSyncForType(ctx, &v1.Pod{}, "pods", coreClient)
	cs.beginSyncForType(ctx, &v1beta1.DaemonSet{}, "daemonsets", extV1beta1Client)
	cs.beginSyncForType(ctx, &v1beta1.Deployment{}, "deployments", extV1beta1Client)
	cs.beginSyncForType(ctx, &v1.ReplicationController{}, "replicationcontrollers", coreClient)
	cs.beginSyncForType(ctx, &v1beta1.ReplicaSet{}, "replicasets", extV1beta1Client)
	cs.beginSyncForType(ctx, &v1.Node{}, "nodes", coreClient)
	cs.beginSyncForType(ctx, &v1.Namespace{}, "namespaces", coreClient)
	cs.beginSyncForType(ctx, &v1.ResourceQuota{}, "resourcequotas", coreClient)
}

func (cs *State) beginSyncForType(ctx context.Context, resType runtime.Object, resName string, client rest.Interface) {
	keysSeen := make(map[interface{}]bool)

	store := k8sutil.FixedFakeCustomStore{
		FakeCustomStore: cache.FakeCustomStore{},
	}
	store.AddFunc = func(obj interface{}) error {
		cs.metricCache.Lock()
		defer cs.metricCache.Unlock()

		if key := cs.metricCache.HandleAdd(obj.(runtime.Object)); key != nil {
			keysSeen[key] = true
		}

		return nil
	}
	store.UpdateFunc = store.AddFunc
	store.DeleteFunc = func(obj interface{}) error {
		cs.metricCache.Lock()
		defer cs.metricCache.Unlock()

		if key := cs.metricCache.HandleDelete(obj.(runtime.Object)); key != nil {
			delete(keysSeen, key)
		}

		return nil
	}
	store.ReplaceFunc = func(list []interface{}, resourceVerion string) error {
		cs.metricCache.Lock()
		defer cs.metricCache.Unlock()

		for k := range keysSeen {
			cs.metricCache.DeleteByKey(k)
			delete(keysSeen, k)
		}
		for i := range list {
			if key := cs.metricCache.HandleAdd(list[i].(runtime.Object)); key != nil {
				keysSeen[key] = true
			}
		}
		return nil
	}

	watchList := cache.NewListWatchFromClient(client, resName, v1.NamespaceAll, fields.Everything())
	cs.reflectors[resName] = cache.NewReflector(watchList, resType, &store, 0)

	go cs.reflectors[resName].Run(ctx.Done())
}

// Stop all running goroutines. There is a bug/limitation in the k8s go
// client's Controller where goroutines are leaked even when using the stop
// channel properly.
// See https://github.com/kubernetes/client-go/blob/release-6.0/tools/cache/controller.go#L144
func (cs *State) Stop() {
	log.Info("Stopping all K8s API resource sync")
	if cs.cancel != nil {
		cs.cancel()
	}
}
