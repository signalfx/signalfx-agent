// Package kubernetes contains an observer that watches the Kubernetes API for
// pods that are running on the same node as the agent.  It uses the streaming
// watch API in K8s so that updates are seen immediately without any polling
// interval.
package kubernetes

import (
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/neo-agent/core/common/kubernetes"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/observers"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

var now = time.Now

const (
	observerType    = "k8s-api"
	nodeEnvVar      = "MY_NODE_NAME"
	namespaceEnvVar = "MY_NAMESPACE"
	runningPhase    = "Running"
)

var logger = log.WithFields(log.Fields{"observerType": observerType})

func init() {
	observers.Register(observerType, func(cbs *observers.ServiceCallbacks) interface{} {
		return &Observer{
			serviceCallbacks: cbs,
		}
	}, &Config{})
}

// Config for Kubernetes API observer
type Config struct {
	config.ObserverConfig
	KubernetesAPI *kubernetes.APIConfig `yaml:"kubernetesAPI" default:"{}"`
}

// Validate the observer-specific config
func (c *Config) Validate() bool {
	if !c.KubernetesAPI.Validate() {
		return false
	}

	if os.Getenv(namespaceEnvVar) == "" {
		logger.Error(fmt.Sprintf("K8s namespace was not provided in the %s envvar",
			namespaceEnvVar))
		return false
	}

	if os.Getenv(nodeEnvVar) == "" {
		logger.Error(fmt.Sprintf("K8s node name was not provided in the %s envvar",
			nodeEnvVar))
		return false
	}
	return true
}

// Observer that watches the Kubernetes API for new pods pertaining to this
// node
type Observer struct {
	config           *Config
	clientset        *k8s.Clientset
	thisNamespace    string
	thisNode         string
	serviceCallbacks *observers.ServiceCallbacks
	stopper          chan struct{}
}

// Configure configures and starts watching for endpoints
func (o *Observer) Configure(config *Config) bool {
	o.thisNamespace = os.Getenv(namespaceEnvVar)
	o.thisNode = os.Getenv(nodeEnvVar)

	var err error
	o.clientset, err = kubernetes.MakeClient(config.KubernetesAPI)
	if err != nil {
		return false
	}

	// Stop previous informers
	if o.stopper != nil {
		o.stopper <- struct{}{}
	}

	o.watchPods()

	return true
}

func (o *Observer) watchPods() {
	o.stopper = make(chan struct{})

	client := o.clientset.Core().RESTClient()
	watchList := cache.NewListWatchFromClient(client, "pods", o.thisNamespace, fields.Everything())

	_, controller := cache.NewInformer(
		watchList,
		&v1.Pod{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				o.changeHandler(nil, obj.(*v1.Pod))
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				o.changeHandler(oldObj.(*v1.Pod), newObj.(*v1.Pod))
			},
			DeleteFunc: func(obj interface{}) {
				o.changeHandler(obj.(*v1.Pod), nil)
			},
		})

	go controller.Run(o.stopper)
}

func (o *Observer) changeHandler(oldPod *v1.Pod, newPod *v1.Pod) {
	// If it is an update, there will be a remove and immediately subsequent
	// add.
	if oldPod != nil && oldPod.Spec.NodeName == o.thisNode {
		endpoints := endpointsInPod(oldPod)
		for i := range endpoints {
			o.serviceCallbacks.Removed(endpoints[i])
		}
	}
	if newPod != nil && newPod.Spec.NodeName == o.thisNode {
		endpoints := endpointsInPod(newPod)
		for i := range endpoints {
			o.serviceCallbacks.Added(endpoints[i])
		}
	}
}

func endpointsInPod(pod *v1.Pod) []services.Endpoint {
	endpoints := make([]services.Endpoint, 0)

	podIP := pod.Status.PodIP
	if pod.Status.Phase != runningPhase {
		return nil
	}

	if len(podIP) == 0 {
		logger.WithFields(log.Fields{
			"podName": pod.Name,
		}).Warn("Pod does not have an IP Address")
		return nil
	}

	for _, container := range pod.Spec.Containers {
		dims := map[string]string{
			"container_name":           container.Name,
			"container_image":          container.Image,
			"kubernetes_pod_name":      pod.Name,
			"kubernetes_pod_namespace": pod.Namespace,
		}
		orchestration := services.NewOrchestration("kubernetes", services.KUBERNETES, dims, services.PRIVATE)

		for _, port := range container.Ports {
			for _, status := range pod.Status.ContainerStatuses {
				// Could possibly be made more efficient by creating maps
				// keyed by name to match up container status and ports.
				if container.Name != status.Name {
					continue
				}

				containerState := "running"
				if status.State.Running == nil {
					// Container is not running.
					continue
				}

				id := fmt.Sprintf("%s-%s-%d", pod.Name, pod.UID[:7], port.ContainerPort)

				endpoint := services.NewEndpointCore(id, port.Name, now(), observerType)
				endpoint.Host = podIP
				endpoint.PortType = services.PortType(port.Protocol)
				endpoint.Port = uint16(port.ContainerPort)

				container := &services.Container{
					ID:        status.ContainerID,
					Names:     []string{status.Name},
					Image:     container.Image,
					Command:   "",
					State:     containerState,
					Labels:    pod.Labels,
					Pod:       pod.Name,
					PodUID:    string(pod.UID),
					Namespace: pod.Namespace,
				}
				endpoints = append(endpoints, &services.ContainerEndpoint{
					EndpointCore:  *endpoint,
					AltPort:       0,
					Container:     *container,
					Orchestration: *orchestration,
				})
			}
		}
	}
	return endpoints
}
