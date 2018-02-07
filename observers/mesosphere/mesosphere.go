package mesosphere

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/observers"
)

var now = time.Now

const (
	observerType = "observers/mesosphere"

	// DefaultPort of the task state api
	DefaultPort = 5051
	// RunningState string for a currently running task
	RunningState = "TASK_RUNNING"
)

// Config for the Mesos observer
type Config struct {
	config.ObserverConfig
	PollIntervalSeconds int    `yaml:"pollIntervalSeconds" default:"10"`
	HostURL             string `yaml:"hostURL"`
}

// Mesosphere observer plugin
type Mesosphere struct {
	config           *Config
	hostURL          string
	client           http.Client
	serviceCallbacks *observers.ServiceCallbacks
	serviceDiffer    *observers.ServiceDiffer
}

// PortMappings for a task
type portMappings []struct {
	HostPort      uint16 `json:"host_port"`
	ContainerPort uint16 `json:"container_port"`
	Protocol      string
}

type tasks struct {
	ID         string
	Hostname   string
	Frameworks []struct {
		ID        string
		Name      string
		Executors []struct {
			Container string
			Tasks     []struct {
				ID     string
				Name   string
				State  string
				Labels []struct {
					Key   string
					Value string
				}
				Discovery struct {
					Name  string
					Ports struct {
						Ports []struct {
							Name     string
							Number   uint16
							Protocol string
							Labels   struct {
								Labels []struct {
									Key   string
									Value string
								}
							}
						}
					}
				}
				Container struct {
					Docker struct {
						Image        string
						Network      string
						PortMappings portMappings `json:"port_mappings"`
					}
				}
			}
		}
	}
}

func init() {
	observers.Register(observerType, func(cbs *observers.ServiceCallbacks) interface{} {
		return &Mesosphere{
			serviceCallbacks: cbs,
		}
	}, &Config{})
}

// Configure the mesosphere observer/client
func (mesos *Mesosphere) Configure(config *Config) error {
	mesos.config = config

	if mesos.config.HostURL == "" {
		hostname, err := os.Hostname()
		if err == nil {
			mesos.config.HostURL = fmt.Sprintf("http://%s:%d", hostname, DefaultPort)
		} else {
			return errors.Wrapf(err, "Could not set default Mesos host URL")
		}
	}

	mesos.client = http.Client{
		Timeout: 10 * time.Second,
	}

	if mesos.serviceDiffer != nil {
		mesos.serviceDiffer.Stop()
	}

	mesos.serviceDiffer = &observers.ServiceDiffer{
		DiscoveryFn:     mesos.discover,
		IntervalSeconds: config.PollIntervalSeconds,
		Callbacks:       mesos.serviceCallbacks,
	}

	mesos.serviceDiffer.Start()

	return nil
}

// Read services from mesosphere
func (mesos *Mesosphere) discover() []services.Endpoint {

	taskInfo, err := mesos.getTasks()
	if err != nil {
		log.WithError(err).Error("Failed to get Mesos tasks")
		return nil
	}

	var instances []services.Endpoint
	for _, framework := range taskInfo.Frameworks {
		for _, executor := range framework.Executors {
			for _, task := range executor.Tasks {

				// only care about running tasks
				if task.State != RunningState {
					continue
				}

				// ports required
				if len(task.Discovery.Ports.Ports) == 0 {
					continue
				}

				var serviceContainer services.Container
				if len(task.Container.Docker.Image) > 0 {
					containerName := fmt.Sprintf("mesos-%s.%s", task.ID, executor.Container)
					containerImage := task.Container.Docker.Image
					containerLabels := map[string]string{}
					for _, label := range task.Labels {
						containerLabels[label.Key] = label.Value
					}
					serviceContainer = services.Container{
						ID:        executor.Container,
						Names:     []string{containerName},
						Image:     containerImage,
						Pod:       "",
						Command:   "",
						State:     "running",
						Labels:    containerLabels,
						Namespace: "",
					}
				} else {
					continue
				}

				orchestration := services.NewOrchestration("mesosphere", services.MESOS, nil, services.PUBLIC)

				for _, port := range task.Discovery.Ports.Ports {
					portName := task.Discovery.Name
					if len(port.Name) > 0 {
						portName = port.Name
					}
					portIP := taskInfo.Hostname
					portType := services.PortType(strings.ToUpper(port.Protocol))
					privatePort := uint16(0)
					publicPort := uint16(port.Number)

					for _, portMapping := range task.Container.Docker.PortMappings {
						if portMapping.HostPort == publicPort {
							privatePort = uint16(portMapping.ContainerPort)
						}
					}

					id := fmt.Sprintf("%s-%d", taskInfo.ID, publicPort)
					endpoint := services.ContainerEndpoint{
						AltPort:       privatePort,
						EndpointCore:  *services.NewEndpointCore(id, portName, observerType),
						Container:     serviceContainer,
						Orchestration: *orchestration,
					}
					endpoint.Host = portIP
					endpoint.PortType = portType
					endpoint.Port = publicPort

					for _, label := range port.Labels.Labels {
						endpoint.PortLabels[label.Key] = label.Value
					}

					endpoint.AddDimension("mesos_agent", taskInfo.ID)
					endpoint.AddDimension("mesos_framework_id", framework.ID)
					endpoint.AddDimension("mesos_framework_name", framework.Name)
					endpoint.AddDimension("mesos_task_id", task.ID)
					endpoint.AddDimension("mesos_task_name", task.Name)

					instances = append(instances, &endpoint)
				}
			}
		}
	}

	return instances
}

// getTasks from mesosphere state api
func (mesos *Mesosphere) getTasks() (*tasks, error) {
	resp, err := mesos.client.Get(fmt.Sprintf("%s/state", mesos.config.HostURL))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get task states: (code=%d, body=%s)",
			resp.StatusCode, body[:512])
	}

	tasks := &tasks{}
	if err := json.Unmarshal(body, tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}
