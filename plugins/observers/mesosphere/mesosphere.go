package mesosphere

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/services"
	"github.com/spf13/viper"
)

const (
	pluginType = "observers/mesosphere"

	// DefaultPort of the task state api
	DefaultPort = 5051
	// RunningState string for a currently running task
	RunningState = "TASK_RUNNING"
)

// Mesosphere observer plugin
type Mesosphere struct {
	plugins.Plugin
	hostURL string
	client  http.Client
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
	plugins.Register(pluginType, NewMesosphere)
}

// NewMesosphere observer
func NewMesosphere(name string, config *viper.Viper) (plugins.IPlugin, error) {
	plugin, err := plugins.NewPlugin(name, pluginType, config)
	if err != nil {
		return nil, err
	}

	return &Mesosphere{plugin, "", http.Client{}}, nil
}

// Configure the mesosphere observer/client
func (mesos *Mesosphere) Configure(config *viper.Viper) error {
	mesos.Config = config
	return mesos.load()
}

func (mesos *Mesosphere) load() error {
	if hostname, err := os.Hostname(); err == nil {
		mesos.Config.SetDefault("hosturl", fmt.Sprintf("http://%s:%d", hostname, DefaultPort))
	}

	hostURL := mesos.Config.GetString("hosturl")
	if len(hostURL) == 0 {
		return errors.New("hostURL config value missing")
	}
	mesos.hostURL = hostURL

	mesos.client = http.Client{
		Timeout: 10 * time.Second,
	}
	return nil
}

// Read services from mesosphere
func (mesos *Mesosphere) Read() (services.Instances, error) {

	taskInfo, err := mesos.getTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %s", err)
	}

	var instances services.Instances
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

				service := services.NewService(task.ID, services.UnknownService, "")

				dims := map[string]string{
					"mesos_agent":          taskInfo.ID,
					"mesos_framework_id":   framework.ID,
					"mesos_framework_name": framework.Name,
					"mesos_task_id":        task.ID,
					"mesos_task_name":      task.Name,
				}

				var serviceContainer *services.Container
				if len(task.Container.Docker.Image) > 0 {
					containerName := fmt.Sprintf("mesos-%s.%s", task.ID, executor.Container)
					containerImage := task.Container.Docker.Image
					dims["container_name"] = containerName
					dims["container_image"] = containerImage
					containerLabels := map[string]string{}
					for _, label := range task.Labels {
						containerLabels[label.Key] = label.Value
					}
					serviceContainer = services.NewContainer(executor.Container, []string{containerName}, containerImage, "", "", "running", containerLabels)
				} else {
					continue
				}

				orchestration := services.NewOrchestration("mesosphere", services.MESOS, dims, services.PUBLIC)

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

					servicePort := services.NewPort(portName, portIP, portType, privatePort, publicPort)
					for _, label := range port.Labels.Labels {
						servicePort.Labels[label.Key] = label.Value
					}

					id := fmt.Sprintf("%s-%d", taskInfo.ID, publicPort)
					si := services.NewInstance(id, service, serviceContainer, orchestration, servicePort, time.Now())
					instances = append(instances, *si)
				}
			}
		}
	}

	sort.Sort(instances)
	return instances, nil
}

// getTasks from mesosphere state api
func (mesos *Mesosphere) getTasks() (*tasks, error) {
	resp, err := mesos.client.Get(fmt.Sprintf("%s/state", mesos.hostURL))
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
