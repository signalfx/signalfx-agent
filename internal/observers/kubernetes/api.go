// Package kubernetes contains an observer that watches the Kubernetes API for
// pods that are running on the same node as the agent.  It uses the streaming
// watch API in K8s so that updates are seen immediately without any polling
// interval.
package kubernetes

import (
	"fmt"
	"os"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/signalfx-agent/internal/core/common/kubernetes"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/core/services"
	"github.com/signalfx/signalfx-agent/internal/observers"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var now = time.Now

const (
	observerType = "k8s-api"
	nodeEnvVar   = "MY_NODE_NAME"
	runningPhase = "Running"
)

var logger = log.WithFields(log.Fields{"observerType": observerType})

func init() {
	observers.Register(observerType, func(cbs *observers.ServiceCallbacks) interface{} {
		return &Observer{
			serviceCallbacks:  cbs,
			endpointsByPodUID: make(map[types.UID][]services.Endpoint),
		}
	}, &Config{})
}

// Config for Kubernetes API observer
type Config struct {
	config.ObserverConfig
	KubernetesAPI *kubernetes.APIConfig `yaml:"kubernetesAPI" default:"{}"`
}

// Validate the observer-specific config
func (c *Config) Validate() error {
	if err := c.KubernetesAPI.Validate(); err != nil {
		return err
	}

	if os.Getenv(nodeEnvVar) == "" {
		return fmt.Errorf("K8s node name was not provided in the %s envvar", nodeEnvVar)
	}
	return nil
}

// Observer that watches the Kubernetes API for new pods pertaining to this
// node
type Observer struct {
	config           *Config
	clientset        *k8s.Clientset
	thisNode         string
	serviceCallbacks *observers.ServiceCallbacks
	// A cache for endpoints so they don't have to be reconstructed when being
	// removed.
	endpointsByPodUID map[types.UID][]services.Endpoint
	stopper           chan struct{}
}

// Configure configures and starts watching for endpoints
func (o *Observer) Configure(config *Config) error {
	// There is a bug/limitation in the k8s go client's Controller where
	// goroutines are leaked even when using the stop channel properly.  So we
	// should avoid going through a shutdown/startup cycle here if nothing is
	// different in the config.
	if reflect.DeepEqual(config, o.config) {
		return nil
	}

	o.thisNode = os.Getenv(nodeEnvVar)

	var err error
	o.clientset, err = kubernetes.MakeClient(config.KubernetesAPI)
	if err != nil {
		return err
	}

	o.stopIfRunning()
	o.watchPods()

	o.config = config

	return nil
}

func (o *Observer) watchPods() {
	o.stopper = make(chan struct{})

	client := o.clientset.Core().RESTClient()
	watchList := cache.NewListWatchFromClient(client, "pods", "", fields.Everything())

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

func (o *Observer) stopIfRunning() {
	// Stop previous informers
	if o.stopper != nil {
		close(o.stopper)
		o.stopper = nil
	}
}

// Handles notifications of changes to pods from the API server
func (o *Observer) changeHandler(oldPod *v1.Pod, newPod *v1.Pod) {
	var newEndpoints []services.Endpoint
	var oldEndpoints []services.Endpoint

	if oldPod != nil && oldPod.Spec.NodeName == o.thisNode {
		oldEndpoints = o.endpointsByPodUID[oldPod.UID]
		delete(o.endpointsByPodUID, oldPod.UID)
	}

	if newPod != nil && newPod.Spec.NodeName == o.thisNode {
		newEndpoints = endpointsInPod(newPod, o.clientset)
		o.endpointsByPodUID[newPod.UID] = newEndpoints
	}

	// Prevent spurious churn of endpoints if they haven't changed
	if reflect.DeepEqual(newEndpoints, oldEndpoints) {
		return
	}

	// If it is an update, there will be a remove and immediately subsequent
	// add.
	for i := range oldEndpoints {
		log.Debugf("Removing K8s endpoint from pod %s", oldPod.UID)
		o.serviceCallbacks.Removed(oldEndpoints[i])
	}

	for i := range newEndpoints {
		log.Debugf("Adding K8s endpoint for pod %s", newPod.UID)
		o.serviceCallbacks.Added(newEndpoints[i])
	}
}

func endpointsInPod(pod *v1.Pod, client *k8s.Clientset) []services.Endpoint {
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

	annotationConfs := annotationsForPod(pod)

	for _, container := range pod.Spec.Containers {
		dims := map[string]string{
			"container_name":           container.Name,
			"container_image":          container.Image,
			"kubernetes_pod_name":      pod.Name,
			"kubernetes_pod_namespace": pod.Namespace,
		}
		orchestration := services.NewOrchestration("kubernetes", services.KUBERNETES, dims, services.PRIVATE)

		var containerState string
		var containerID string
		var containerName string

		for _, status := range pod.Status.ContainerStatuses {
			if container.Name != status.Name {
				continue
			}

			if status.State.Running == nil {
				break
			}
			containerState = "running"
			containerID = status.ContainerID
			containerName = status.Name
		}

		if containerState != "running" {
			continue
		}

		for _, port := range container.Ports {
			id := fmt.Sprintf("%s-%s-%d", pod.Name, pod.UID[:7], port.ContainerPort)

			endpoint := services.NewEndpointCore(id, port.Name, observerType)

			portAnnotations := annotationConfs.FilterByPortOrPortName(port.ContainerPort, port.Name)
			monitorType, extraConf, err := configFromAnnotations(container.Name, portAnnotations, pod, client)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("K8s port has invalid config annotations")
			} else {
				endpoint.Configuration = extraConf
				endpoint.MonitorType = monitorType
			}

			endpoint.Host = podIP
			endpoint.PortType = services.PortType(port.Protocol)
			endpoint.Port = uint16(port.ContainerPort)

			container := &services.Container{
				ID:        containerID,
				Names:     []string{containerName},
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
	return endpoints
}

// Shutdown the service differ routine
func (o *Observer) Shutdown() {
	o.stopIfRunning()
	o.config = nil
}
