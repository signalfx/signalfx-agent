package kubernetes

import (
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"time"

	"fmt"

	"errors"

	"encoding/json"

	"crypto/tls"

	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/services"
	"github.com/spf13/viper"
)

var now = time.Now

// phase Kubernetes pod phase
type phase string

const (
	// RunningPhase Kubernetes running phase
	runningPhase phase = "Running"
)

// Kubernetes observer plugin
type Kubernetes struct {
	plugins.Plugin
	hostURL string
}

// pod structure from kubelet
type pods struct {
	Items []struct {
		Metadata struct {
			Name      string
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

// NewKubernetes constructor
func NewKubernetes(name string, config *viper.Viper) (*Kubernetes, error) {
	plugin, err := plugins.NewPlugin(name, config)
	if err != nil {
		return nil, err
	}

	hostURL := config.GetString("hosturl")
	if len(hostURL) == 0 {
		return nil, errors.New("hostURL config value missing")
	}
	return &Kubernetes{plugin, hostURL}, nil
}

// Map adds additional data from the kubelet into instances
func (k *Kubernetes) Map(sis services.Instances) (services.Instances, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: k.Config.GetBool("ignoretlsverify"),
		},
	}
	client := http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}
	resp, err := client.Get(fmt.Sprintf("%s/pods", k.hostURL))
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
	pods, err := load(body)
	if err != nil {
		return nil, fmt.Errorf("failed to load pods list: %s", err)
	}

	// Map the pods list into given service instances.
	mapped, err := k.doMap(sis, pods)
	if err != nil {
		return nil, fmt.Errorf("failed to map pods list: %s", err)
	}

	return mapped, nil

}

// doMap takes a list of service instance and applies information discovered
// from Kubernetes for matching containers
func (k *Kubernetes) doMap(sis services.Instances, pods *pods) (services.Instances, error) {
	var instances services.Instances

	for _, pod := range pods.Items {
		podIP := pod.Status.PodIP
		if pod.Status.Phase != runningPhase {
			continue
		}

		if len(podIP) == 0 {
			log.Printf("error: %s missing pod IP", pod.Metadata.Name)
			continue
		}

		for _, container := range pod.Spec.Containers {
			dims := map[string]string{
				"container_name":           container.Name,
				"container_image":          container.Image,
				"kubernetes_pod_name":      pod.Metadata.Name,
				"kubernetes_pod_namespace": pod.Metadata.Namespace,
				"kubernetes_node":          pod.Spec.NodeName,
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
					if _, ok := status.State[containerState]; !ok {
						// Container is not running.
						continue
					}

					id := fmt.Sprintf("%s-%s-%d", k.String(), pod.Metadata.Name, port.ContainerPort)
					service := services.NewService(id, services.UnknownService)
					servicePort := services.NewPort(podIP, port.Protocol, port.ContainerPort, 0)
					container := services.NewContainer(status.ContainerID,
						[]string{status.Name}, container.Image, pod.Metadata.Name, "",
						containerState, pod.Metadata.Labels)
					instances = append(instances, *services.NewInstance(id, service, container,
						orchestration, servicePort, now()))
				}
			}
		}
	}

	sort.Sort(instances)
	return instances, nil
}

func load(body []byte) (*pods, error) {
	pods := &pods{}
	if err := json.Unmarshal(body, pods); err != nil {
		return nil, err
	}

	return pods, nil
}
