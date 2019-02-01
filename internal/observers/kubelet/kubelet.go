package kubelet

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/signalfx-agent/internal/core/common/kubelet"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/core/services"
	"github.com/signalfx/signalfx-agent/internal/observers"
)

var now = time.Now

// phase is the pod's phase
type phase string

const (
	observerType = "k8s-kubelet"
	// RunningPhase running phase
	runningPhase phase = "Running"
)

// OBSERVER(k8s-kubelet): Discovers service endpoints running on the same node
// as the agent by querying the local kubelet instance.  It is generally
// recommended to use the [k8s-api](./k8s-api.md) observer because
// authentication to the local kubelet can be more difficult to setup, and also
// the kubelet API is technically not documented for public consumption, so
// this observer may break more easily in future K8s versions.

// ENDPOINT_TYPE(ContainerEndpoint): true

// DIMENSION(kubernetes_namespace): The namespace that the discovered service
// endpoint is running in.

// DIMENSION(kubernetes_pod_name): The name of the running pod that is exposing
// the discovered endpoint

// DIMENSION(kubernetes_pod_uid): The UID of the pod that is exposing the
// discovered endpoint

// DIMENSION(container_spec_name): The short name of the container in the pod spec,
// **NOT** the running container's name in the Docker engine

var logger = log.WithFields(log.Fields{"observerType": observerType})

// Config for Kubelet observer
type Config struct {
	config.ObserverConfig
	// How often to poll the Kubelet instance for pod information
	PollIntervalSeconds int `yaml:"pollIntervalSeconds" default:"10"`
	// Config for the Kubelet HTTP client
	KubeletAPI kubelet.APIConfig `yaml:"kubeletAPI" default:"{}"`
}

// Validate the observer-specific config
func (c *Config) Validate() error {
	if c.PollIntervalSeconds < 1 {
		return errors.New("pollIntervalSeconds must be greater than 0")
	}

	if (c.KubeletAPI.CACertPath != "" ||
		c.KubeletAPI.ClientCertPath != "" ||
		c.KubeletAPI.ClientKeyPath != "") &&
		c.KubeletAPI.AuthType != kubelet.AuthTypeTLS {
		logger.WithFields(log.Fields{
			"kubeletAuthType": c.KubeletAPI.AuthType,
		}).Warn("Kubelet TLS client auth config keys are set while authType is not 'tls'")
		// Does not render invalid, but warn user nonetheless
	}

	return nil
}

// Observer for kubelet
type Observer struct {
	config           *Config
	client           *kubelet.Client
	serviceDiffer    *observers.ServiceDiffer
	serviceCallbacks *observers.ServiceCallbacks
}

// pod structure from kubelet
type pods struct {
	Items []struct {
		Metadata struct {
			Name      string
			UID       string `json:"uid,omitempty"`
			Namespace string
			Labels    map[string]string
		}
		Spec struct {
			NodeName   string
			Containers []struct {
				Name  string
				Image string
				Ports []struct {
					Name          string
					ContainerPort uint16
					Protocol      services.PortType
				}
			}
		}
		Status struct {
			Phase             phase
			PodIP             string
			ContainerStatuses []struct {
				Name        string
				ContainerID string
				State       map[string]struct{}
			}
		}
	}
}

func init() {
	observers.Register(observerType, func(cbs *observers.ServiceCallbacks) interface{} {
		return &Observer{
			serviceCallbacks: cbs,
		}
	}, &Config{})
}

// Configure the kubernetes observer/client
func (k *Observer) Configure(config *Config) error {
	var err error
	k.client, err = kubelet.NewClient(&config.KubeletAPI)
	if err != nil {
		return err
	}

	if k.serviceDiffer != nil {
		k.serviceDiffer.Stop()
	}

	k.serviceDiffer = &observers.ServiceDiffer{
		DiscoveryFn:     k.discover,
		IntervalSeconds: config.PollIntervalSeconds,
		Callbacks:       k.serviceCallbacks,
	}
	k.config = config

	k.serviceDiffer.Start()

	return nil
}

// Map adds additional data from the kubelet into instances
func (k *Observer) getPods() (*pods, error) {
	resp, err := k.client.Get(fmt.Sprintf("%s/pods", k.config.KubeletAPI.URL))
	if err != nil {
		return nil, fmt.Errorf("kubelet request failed: %s", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get /pods: (code=%d, body=%s)",
			resp.StatusCode, body[:512])
	}

	// Load pods list.
	pods, err := loadJSON(body)
	if err != nil {
		return nil, fmt.Errorf("failed to load pods list: %s", err)
	}
	return pods, nil
}

func loadJSON(body []byte) (*pods, error) {
	pods := &pods{}
	if err := json.Unmarshal(body, pods); err != nil {
		return nil, err
	}

	return pods, nil
}

func (k *Observer) discover() []services.Endpoint {
	var instances []services.Endpoint

	pods, err := k.getPods()
	if err != nil {
		logger.WithFields(log.Fields{
			"error":      err,
			"kubeletURL": k.config.KubeletAPI.URL,
		}).Error("Could not get pods from Kubelet API")
		return nil
	}

	for _, pod := range pods.Items {
		podIP := pod.Status.PodIP
		if pod.Status.Phase != runningPhase {
			continue
		}

		if len(podIP) == 0 {
			logger.WithFields(log.Fields{
				"podName": pod.Metadata.Name,
			}).Warn("Pod does not have an IP Address")
			continue
		}

		for _, container := range pod.Spec.Containers {
			dims := map[string]string{
				"container_spec_name":  container.Name,
				"kubernetes_pod_name":  pod.Metadata.Name,
				"kubernetes_pod_uid":   pod.Metadata.UID,
				"kubernetes_namespace": pod.Metadata.Namespace,
			}
			orchestration := services.NewOrchestration("kubernetes", services.KUBERNETES, services.PRIVATE)

			for _, port := range container.Ports {
				for _, status := range pod.Status.ContainerStatuses {
					// Could possibly be made more efficient by creating maps
					// keyed by name to match up container status and ports.
					if container.Name != status.Name {
						continue
					}

					containerState := "running"
					if _, ok := status.State[containerState]; !ok {
						// Container is not running.
						continue
					}

					id := fmt.Sprintf("%s-%s-%d", pod.Metadata.Name, pod.Metadata.UID[:7], port.ContainerPort)

					endpoint := services.NewEndpointCore(id, port.Name, observerType, dims)
					endpoint.Host = podIP
					endpoint.PortType = port.Protocol
					endpoint.Port = port.ContainerPort

					container := &services.Container{
						ID:      status.ContainerID,
						Names:   []string{status.Name},
						Image:   container.Image,
						Command: "",
						State:   containerState,
						Labels:  pod.Metadata.Labels,
					}
					instances = append(instances, &services.ContainerEndpoint{
						EndpointCore:  *endpoint,
						AltPort:       0,
						Container:     *container,
						Orchestration: *orchestration,
					})
				}
			}
		}
	}

	return instances
}

// Shutdown the service differ routine
func (k *Observer) Shutdown() {
	if k.serviceDiffer != nil {
		k.serviceDiffer.Stop()
	}
}
