package kubernetes

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/observers"
)

var now = time.Now

// phase Kubernetes pod phase
type phase string

const (
	observerType = "kubernetes"
	// RunningPhase Kubernetes running phase
	runningPhase phase = "Running"
)

var logger = log.WithFields(log.Fields{"observerType": observerType})

type AuthType string

const (
	AuthTypeNone AuthType = "none"
	AuthTypeTLS  AuthType = "tls"
)

type Config struct {
	config.ObserverConfig
	PollIntervalSeconds int `default:"10"`
	KubeletAPI          struct {
		URL            string   `yaml:"url,omitempty"`
		AuthType       AuthType `default: "none"`
		SkipVerify     bool     `default: "false"`
		CACertPath     string   `yaml:"caCertPath,omitempty"`
		ClientCertPath string
		ClientKeyPath  string
	}
}

// Kubernetes observer plugin
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
					Protocol      observers.PortType
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

	if (config.KubeletAPI.CACertPath != "" ||
		config.KubeletAPI.ClientCertPath != "" ||
		config.KubeletAPI.ClientKeyPath != "") &&
		config.KubeletAPI.AuthType != AuthTypeTLS {
		logger.WithFields(log.Fields{
			"kubeletAuthType": config.KubeletAPI.AuthType,
		}).Warn("Kubelet TLS client auth config keys are set while authType is not 'tls'")
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

	if config.PollIntervalSeconds < 1 {
		logger.WithFields(log.Fields{
			"pollIntervalSeconds": config.PollIntervalSeconds,
		}).Error("pollIntervalSeconds must be greater than 0")
		return false
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
	pods, err := loadJson(body)
	if err != nil {
		return nil, fmt.Errorf("failed to load pods list: %s", err)
	}
	return pods, nil
}

func loadJson(body []byte) (*pods, error) {
	pods := &pods{}
	if err := json.Unmarshal(body, pods); err != nil {
		return nil, err
	}

	return pods, nil
}

func (k *Kubernetes) discover() []*observers.ServiceInstance {
	var instances []*observers.ServiceInstance

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
			orchestration := observers.NewOrchestration("kubernetes", observers.KUBERNETES, dims, observers.PRIVATE)

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

					id := fmt.Sprintf("%p-%s-%d", k, pod.Metadata.Name, port.ContainerPort)
					servicePort := observers.NewPort(port.Name, podIP, port.Protocol, port.ContainerPort, 0)
					container := observers.NewContainer(status.ContainerID,
						[]string{status.Name}, container.Image, pod.Metadata.Name, "",
						containerState, pod.Metadata.Labels, pod.Metadata.Namespace)
					instances = append(instances, observers.NewServiceInstance(id, container,
						orchestration, servicePort, now()))
				}
			}
		}
	}

	return instances
}
