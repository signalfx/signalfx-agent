package testhelpers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/signalfx-agent/internal/utils"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	//"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apimachinery/pkg/runtime"
)

// ResourceType is an enum
type ResourceType int

// Enum values for ResourceType
const (
	Pods ResourceType = iota
	Deployments
	ReplicationControllers
	DaemonSets
	ReplicaSets
	Secrets
)

// FakeK8s is a mock K8s API server.  It can serve both list and watch
// requests.
type FakeK8s struct {
	server *httptest.Server
	// Resources that have been inserted on the ResourceInput channel
	state      map[ResourceType]map[types.UID]runtime.Object
	secrets    map[string]*v1.Secret
	stateMutex sync.Mutex
	// Used by tests to insert resources for test
	EventInput chan WatchEvent
	// Channels to send new resources to watchers (we only support one watcher
	// per resource)
	subs      map[ResourceType]chan WatchEvent
	subsMutex sync.Mutex
	// Stops the resource accepter goroutine
	eventStopper chan struct{}
	// Stops all of the watchers
	stoppers map[ResourceType]chan struct{}
}

// NewFakeK8s makes a new FakeK8s
func NewFakeK8s() *FakeK8s {
	return &FakeK8s{
		state:      make(map[ResourceType]map[types.UID]runtime.Object),
		secrets:    make(map[string]*v1.Secret),
		EventInput: make(chan WatchEvent),
		subs:       make(map[ResourceType]chan WatchEvent),
		stoppers:   make(map[ResourceType]chan struct{}),
	}
}

// Start creates the server and starts it
func (f *FakeK8s) Start() {
	f.server = httptest.NewUnstartedServer(f)
	f.server.StartTLS()

	f.eventStopper = make(chan struct{})
	go f.acceptEvents(f.eventStopper)
}

// Close stops the server and all watchers
func (f *FakeK8s) Close() {
	close(f.eventStopper)
	for _, ch := range f.stoppers {
		close(ch)
	}

	f.server.Listener.Close()
	f.server.Close()

}

// URL is the of the mock server to point your objects under test to
func (f *FakeK8s) URL() string {
	return f.server.URL
}

// SetInitialList adds resources to the server state that are served when doing
// list requests.  l can be a list of any of the supported resource types.
func (f *FakeK8s) SetInitialList(l interface{}) {
	var resType ResourceType
	// Trying to do this more generically locks up on the type assertion from
	// interface{} without errors, not sure why
	switch v := l.(type) {
	case []*v1.Pod:
		resType = Pods
		for _, r := range v {
			f.addToState(resType, r.UID, r)
		}
	case []*v1beta1.Deployment:
		resType = Deployments
		for _, r := range v {
			f.addToState(resType, r.UID, r)
		}
	case []*v1beta1.ReplicaSet:
		resType = ReplicaSets
		for _, r := range v {
			f.addToState(resType, r.UID, r)
		}
	case []*v1beta1.DaemonSet:
		resType = DaemonSets
		for _, r := range v {
			f.addToState(resType, r.UID, r)
		}
	case []*v1.ReplicationController:
		resType = ReplicationControllers
		for _, r := range v {
			f.addToState(resType, r.UID, r)
		}
	default:
		panic("Unsupported resource type!")
	}
}

func (f *FakeK8s) acceptEvents(stopper <-chan struct{}) {
	for {
		select {
		case <-stopper:
			return
		case e := <-f.EventInput:
			var resType ResourceType
			var uid types.UID

			switch v := e.Object.(type) {
			case *v1.Pod:
				resType = Pods
				uid = v.UID
			case *v1.ReplicationController:
				resType = ReplicationControllers
				uid = v.UID
			case *v1beta1.Deployment:
				resType = Deployments
				uid = v.UID
			case *v1beta1.DaemonSet:
				resType = DaemonSets
				uid = v.UID
			case *v1beta1.ReplicaSet:
				resType = ReplicaSets
				uid = v.UID
			default:
				log.Printf("Unknown resource type for %#v", e)
				continue
			}

			f.addToState(resType, uid, e.Object)
			// Send it out to any watchers
			if f.subs[resType] != nil {
				f.subs[resType] <- e
			}
		}
	}
}

func (f *FakeK8s) addToState(resType ResourceType, uid types.UID, resource runtime.Object) {
	f.stateMutex.Lock()
	defer f.stateMutex.Unlock()

	if f.state[resType] == nil {
		f.state[resType] = make(map[types.UID]runtime.Object, 0)
	}
	f.state[resType][uid] = resource
}

// AddSecret accepts a secret to serve later.
func (f *FakeK8s) AddSecret(secret *v1.Secret) {
	f.secrets[secret.Namespace+"/"+secret.Name] = secret
}

func (f *FakeK8s) handleGetSecret(params map[string]string, rw http.ResponseWriter) {
	if secret, ok := f.secrets[params["namespace"]+"/"+params["name"]]; ok {
		s, _ := json.Marshal(secret)
		rw.Write(s)
		return
	}

	log.Errorf("Secret %s/%s not found", params["namespace"], params["name"])
	rw.WriteHeader(http.StatusNotFound)
}

// ServeHTTP handles a single request
func (f *FakeK8s) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	log.Printf("Request: %s", r.URL.String())

	rw.Header().Add("Content-Type", "application/json")

	for reText, handler := range map[string]func(map[string]string, http.ResponseWriter){
		`/api/v1/namespaces/(?P<namespace>\w+)/secrets/(?P<name>\w+)`: f.handleGetSecret,
	} {
		re := regexp.MustCompile(reText)
		groupMap := utils.RegexpGroupMap(re, r.URL.Path)
		if groupMap != nil {
			handler(groupMap, rw)
			return
		}
	}

	var resource ResourceType
	var isWatch = false
	switch r.URL.Path {
	case "/api/v1/watch/pods":
		isWatch = true
		fallthrough
	case "/api/v1/pods":
		resource = Pods
	case "/api/v1/watch/replicationcontrollers":
		isWatch = true
		fallthrough
	case "/api/v1/replicationcontrollers":
		resource = ReplicationControllers
	case "/apis/extensions/v1beta1/watch/replicasets":
		isWatch = true
		fallthrough
	case "/apis/extensions/v1beta1/replicasets":
		resource = ReplicaSets
	case "/apis/extensions/v1beta1/watch/daemonsets":
		isWatch = true
		fallthrough
	case "/apis/extensions/v1beta1/daemonsets":
		resource = DaemonSets
	case "/apis/extensions/v1beta1/watch/deployments":
		isWatch = true
		fallthrough
	case "/apis/extensions/v1beta1/deployments":
		resource = Deployments
	default:
		log.Printf("API Resource Not Implemented: %s", r.URL.String())
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	if isWatch {
		rw.Header().Add("Transfer-Encoding", "chunked")
		f.stoppers[resource] = make(chan struct{})
		// This must block in order to continue to be able to write to the
		// ResponseWriter
		f.startWatcher(resource, rw, f.stoppers[resource])
	} else {
		f.sendList(resource, rw)
	}

	log.Print("Done with request: ", r.URL.String())
}

// Start a long running routine that will send everything received on the
// `EventInput` channel as JSON back to the client
func (f *FakeK8s) startWatcher(resType ResourceType, rw http.ResponseWriter, stopper <-chan struct{}) {
	// There could be multiple watchers starting simultaneously
	f.subsMutex.Lock()

	if f.subs[resType] != nil {
		panic("We don't support more than one watcher at a time!")
	}

	eventCh := make(chan WatchEvent)
	f.subs[resType] = eventCh

	f.subsMutex.Unlock()

	for {
		select {
		case r := <-eventCh:
			d, _ := json.Marshal(r)
			rw.Write(d)
			rw.Write([]byte("\n"))
			rw.(http.Flusher).Flush()
		case <-stopper:
			return
		}
	}
}

func (f *FakeK8s) sendList(resType ResourceType, rw http.ResponseWriter) {
	items := make([]runtime.RawExtension, 0)
	for _, i := range f.state[resType] {
		items = append(items, runtime.RawExtension{
			Object: i,
		})
	}

	l := v1.List{
		TypeMeta: typeMeta(resType),
		ListMeta: metav1.ListMeta{},
		Items:    items,
	}

	d, _ := json.Marshal(l)
	log.Print("list: ", string(d))

	rw.Write(d)
}

func typeMeta(rt ResourceType) metav1.TypeMeta {
	switch rt {
	case Pods:
		return metav1.TypeMeta{Kind: "PodList", APIVersion: "v1"}
	case ReplicationControllers:
		return metav1.TypeMeta{Kind: "ReplicationControllerList", APIVersion: "v1"}
	case Deployments:
		return metav1.TypeMeta{Kind: "DeploymentList", APIVersion: "extensions/v1beta1"}
	case DaemonSets:
		return metav1.TypeMeta{Kind: "DaemonSetList", APIVersion: "extensions/v1beta1"}
	case ReplicaSets:
		return metav1.TypeMeta{Kind: "ReplicaSetList", APIVersion: "extensions/v1beta1"}
	default:
		panic("Unknown resource type: " + string(rt))
	}
}
