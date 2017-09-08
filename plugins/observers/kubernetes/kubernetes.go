package kubernetes

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"fmt"

	"errors"

	"encoding/json"

	"crypto/tls"
	"crypto/x509"

	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/services"
	"github.com/spf13/viper"
)

var now = time.Now

// phase Kubernetes pod phase
type phase string

const (
	pluginType = "observers/kubernetes"
	// RunningPhase Kubernetes running phase
	runningPhase phase = "Running"
	// DefaultPort of kubelet
	DefaultPort = 10250
)

// Kubernetes observer plugin
type Kubernetes struct {
	plugins.Plugin
	hostURL string
	client  http.Client
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
	plugins.Register(pluginType, NewKubernetes)
}

// NewKubernetes constructor
func NewKubernetes(name string, config *viper.Viper) (plugins.IPlugin, error) {
	plugin, err := plugins.NewPlugin(name, pluginType, config)
	if err != nil {
		return nil, err
	}

	return &Kubernetes{plugin, "", http.Client{}}, nil
}

// Configure the kubernetes observer/client
func (k *Kubernetes) Configure(config *viper.Viper) error {
	k.Config = config
	return k.load()
}

func (k *Kubernetes) load() error {
	var hostname string
	if k.Config.GetString("host") != "" {
		hostname = k.Config.GetString("host")
	} else {
		var err error
		hostname, err = os.Hostname()
		if err != nil {
			hostname = "localhost"
		}
	}

	k.Config.SetDefault("hosturl", fmt.Sprintf("https://%s:%d", hostname, DefaultPort))

	hostURL := k.Config.GetString("hosturl")
	if len(hostURL) == 0 {
		return errors.New("hostURL config value missing")
	}
	k.hostURL = hostURL

	skipVerify := k.Config.GetBool("tls.skipVerify")
	caCert := k.Config.GetString("tls.caCert")
	clientCert := k.Config.GetString("tls.clientCert")
	clientKey := k.Config.GetString("tls.clientKey")

	certs, err := x509.SystemCertPool()
	if err != nil {
		return err
	}

	if caCert != "" {
		bytes, err := ioutil.ReadFile(caCert)
		if err != nil {
			return fmt.Errorf("unable to read CA certificate: %s", err)
		}
		if !certs.AppendCertsFromPEM(bytes) {
			return fmt.Errorf("unable to add %s to certs", caCert)
		}
		log.Printf("configured TLS cert from %s", caCert)
	}

	var clientCerts []tls.Certificate

	if clientCert != "" && clientKey != "" {
		cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
		if err != nil {
			return err
		}
		clientCerts = append(clientCerts, cert)
		log.Printf("configured TLS client cert %s with key %s", clientCert, clientKey)
	}

	tlsConfig := &tls.Config{
		Certificates:       clientCerts,
		InsecureSkipVerify: skipVerify,
		RootCAs:            certs,
	}
	tlsConfig.BuildNameToCertificate()

	log.Printf("configured InsecureSkipVerify=%t", skipVerify)

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	k.client = http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}
	return nil
}

// Map adds additional data from the kubelet into instances
func (k *Kubernetes) Map(sis services.Instances) (services.Instances, error) {
	resp, err := k.client.Get(fmt.Sprintf("%s/pods", k.hostURL))
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
				"container_name":       container.Name,
				"container_image":      container.Image,
				"kubernetes_pod_name":  pod.Metadata.Name,
				"kubernetes_namespace": pod.Metadata.Namespace,
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
					service := services.NewService(pod.Metadata.Name, services.UnknownService, "")
					servicePort := services.NewPort(port.Name, podIP, port.Protocol, port.ContainerPort, 0)
					container := services.NewContainer(status.ContainerID,
						[]string{status.Name}, container.Image, pod.Metadata.Name, "",
						containerState, pod.Metadata.Labels, pod.Metadata.Namespace)
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
