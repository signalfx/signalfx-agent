package kubelet

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/observers"
)

var now = time.Now

// phase Kubernetes pod phase
type phase string

const (
	observerType = "k8s-kubelet"
	// RunningPhase Kubernetes running phase
	runningPhase phase = "Running"
)

var logger = log.WithFields(log.Fields{"observerType": observerType})

// AuthType to use when connecting to kubelet
type AuthType string

const (
	// AuthTypeNone means there is no authentication to kubelet
	AuthTypeNone AuthType = "none"
	// AuthTypeTLS indicates that client TLS auth is desired
	AuthTypeTLS AuthType = "tls"
)

// Config for Kubernetes observer
type Config struct {
	config.ObserverConfig
	PollIntervalSeconds int `yaml:"pollIntervalSeconds" default:"10"`
	KubeletAPI          struct {
		URL            string   `yaml:"url"`
		AuthType       AuthType `yaml:"authType" default:"none"`
		SkipVerify     bool     `yaml:"skipVerify" default:"false"`
		CACertPath     string   `yaml:"caCertPath"`
		ClientCertPath string   `yaml:"clientCertPath"`
		ClientKeyPath  string   `yaml:"clientKeyPath"`
	} `yaml:"kubeletAPI" default:"{}"`
}

// Validate the observer-specific config
func (c *Config) Validate() bool {
	if c.PollIntervalSeconds < 1 {
		logger.WithFields(log.Fields{
			"pollIntervalSeconds": c.PollIntervalSeconds,
		}).Error("pollIntervalSeconds must be greater than 0")
		return false
	}

	if (c.KubeletAPI.CACertPath != "" ||
		c.KubeletAPI.ClientCertPath != "" ||
		c.KubeletAPI.ClientKeyPath != "") &&
		c.KubeletAPI.AuthType != AuthTypeTLS {
		logger.WithFields(log.Fields{
			"kubeletAuthType": c.KubeletAPI.AuthType,
		}).Warn("Kubelet TLS client auth config keys are set while authType is not 'tls'")
		// Does not render invalid, but warn user nonetheless
	}

	return true
}

// Kubernetes observer
type Kubernetes struct {
	config           *Config
	client           http.Client
	serviceDiffer    *observers.ServiceDiffer
	serviceCallbacks *observers.ServiceCallbacks
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

func init() {
	observers.Register(observerType, func(cbs *observers.ServiceCallbacks) interface{} {
		return &Kubernetes{
			serviceCallbacks: cbs,
		}
	}, &Config{})
}

// Configure the kubernetes observer/client
func (k *Kubernetes) Configure(config *Config) bool {
	if config.KubeletAPI.URL == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "localhost"
		}
		config.KubeletAPI.URL = fmt.Sprintf("https://%s:10250", hostname)
	}

	certs, err := x509.SystemCertPool()
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err,
		}).Error("Could not get TLS system CA list")
		return false
	}

	if config.KubeletAPI.CACertPath != "" {
		bytes, err := ioutil.ReadFile(config.KubeletAPI.CACertPath)
		if err != nil {
			logger.WithFields(log.Fields{
				"error":      err,
				"caCertPath": config.KubeletAPI.CACertPath,
			}).Error("CA cert path could not be read")
			return false
		}
		if !certs.AppendCertsFromPEM(bytes) {
			logger.WithFields(log.Fields{
				"error":      err,
				"caCertPath": config.KubeletAPI.CACertPath,
			}).Error("CA cert file is not the right format")
			return false
		}
	}

	var clientCerts []tls.Certificate

	clientCertPath := config.KubeletAPI.ClientCertPath
	clientKeyPath := config.KubeletAPI.ClientKeyPath
	if clientCertPath != "" && clientKeyPath != "" {
		cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
		if err != nil {
			logger.WithFields(log.Fields{
				"error":          err,
				"clientKeyPath":  clientKeyPath,
				"clientCertPath": clientCertPath,
			}).Error("Kubelet client cert/key could not be loaded")
			return false
		}
		clientCerts = append(clientCerts, cert)
		logger.Infof("Configured TLS client cert in %s with key %s", clientCertPath, clientKeyPath)
	}

	tlsConfig := &tls.Config{
		Certificates:       clientCerts,
		InsecureSkipVerify: config.KubeletAPI.SkipVerify,
		RootCAs:            certs,
	}
	tlsConfig.BuildNameToCertificate()

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	k.client = http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
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

	return true
}

// Map adds additional data from the kubelet into instances
func (k *Kubernetes) getPods() (*pods, error) {
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

func (k *Kubernetes) discover() []services.Endpoint {
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
				"container_name":           container.Name,
				"container_image":          container.Image,
				"kubernetes_pod_name":      pod.Metadata.Name,
				"kubernetes_pod_namespace": pod.Metadata.Namespace,
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

					id := fmt.Sprintf("%s-%s-%d", pod.Metadata.Name, status.ContainerID[:12], port.ContainerPort)

					endpoint := services.NewEndpointCore(id, port.Name, now(), observerType)
					endpoint.Host = podIP
					endpoint.PortType = port.Protocol
					endpoint.Port = port.ContainerPort

					container := &services.Container{
						ID:        status.ContainerID,
						Names:     []string{status.Name},
						Image:     container.Image,
						Command:   "",
						State:     containerState,
						Labels:    pod.Metadata.Labels,
						Pod:       pod.Metadata.Name,
						Namespace: pod.Metadata.Namespace,
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
func (k *Kubernetes) Shutdown() {
	if k.serviceDiffer != nil {
		k.serviceDiffer.Stop()
	}
}
