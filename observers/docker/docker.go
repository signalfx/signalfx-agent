package docker

import (
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/observers"
)

const (
	observerType     = "docker"
	dockerAPIVersion = "v1.22"
)

var logger = log.WithFields(log.Fields{"observerType": observerType})

// Docker observer plugin
type Docker struct {
	client           *client.Client
	serviceCallbacks *observers.ServiceCallbacks
	serviceDiffer    *observers.ServiceDiffer
	config           *Config
}

type Config struct {
	config.ObserverConfig
	DockerURL string `default:"unix:///var/run/docker.sock"`
	// How often to poll the docker API
	PollIntervalSeconds int `default:"10"`
}

func init() {
	observers.Register(observerType, func(cbs *observers.ServiceCallbacks) interface{} {
		return &Docker{
			serviceCallbacks: cbs,
		}
	}, &Config{})
}

// Configure the docker client
func (docker *Docker) Configure(config *Config) bool {
	defaultHeaders := map[string]string{"User-Agent": "signalfx-agent"}

	var err error
	docker.client, err = client.NewClient(config.DockerURL, dockerAPIVersion, nil, defaultHeaders)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Could not create docker client")
		return false
	}

	if config.PollIntervalSeconds < 1 {
		logger.WithFields(log.Fields{
			"pollIntervalSeconds": config.PollIntervalSeconds,
		}).Error("pollIntervalSeconds must be greater than 0")
		return false
	}

	docker.serviceDiffer = &observers.ServiceDiffer{
		DiscoveryFn:     docker.discover,
		IntervalSeconds: config.PollIntervalSeconds,
		Callbacks:       docker.serviceCallbacks,
	}
	docker.config = config

	docker.serviceDiffer.Start()

	return true
}

// Discover services by querying docker api
func (docker *Docker) discover() []*observers.ServiceInstance {
	options := types.ContainerListOptions{All: true}
	containers, err := docker.client.ContainerList(context.Background(), options)
	if err != nil {
		logger.WithFields(log.Fields{
			"options":   options,
			"dockerURL": docker.config.DockerURL,
			"error":     err,
		}).Error("Could not get container list from docker")
		return nil
	}

	instances := make([]*observers.ServiceInstance, 0)

	for _, c := range containers {
		if c.State == "running" {
			serviceContainer := observers.NewContainer(c.ID, c.Names, c.Image, "", c.Command, c.State, c.Labels, "")
			for _, port := range c.Ports {
				servicePort := observers.NewPort("", "127.0.0.1", observers.PortType(port.Type), uint16(port.PrivatePort), uint16(port.PublicPort))

				id := serviceContainer.PrimaryName() + "-" + c.ID[:12] + "-" + strconv.Itoa(port.PrivatePort)

				dims := map[string]string{
					"container_name":  serviceContainer.PrimaryName(),
					"container_image": c.Image,
				}

				orchestration := observers.NewOrchestration("docker", observers.DOCKER, dims, observers.PUBLIC)

				si := observers.NewServiceInstance(id, serviceContainer, orchestration, servicePort, time.Now())

				instances = append(instances, si)
			}
		}
	}

	return instances
}

func (docker *Docker) Shutdown() {
	docker.serviceDiffer.Stop()
}
