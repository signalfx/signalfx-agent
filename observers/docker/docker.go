// Package docker is an observer that watches a docker daemon and reports
// container ports as service endpoints.
package docker

import (
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
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

// Config specific to the Docker observer
type Config struct {
	config.ObserverConfig
	DockerURL string `yaml:"dockerURL" default:"unix:///var/run/docker.sock"`
	// How often to poll the docker API
	PollIntervalSeconds int `default:"10"`
}

// Validate the docker-specific config
func (c *Config) Validate() bool {
	if c.PollIntervalSeconds < 1 {
		logger.WithFields(log.Fields{
			"pollIntervalSeconds": c.PollIntervalSeconds,
		}).Error("pollIntervalSeconds must be greater than 0")
		return false
	}
	return true
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

	if docker.serviceDiffer != nil {
		docker.serviceDiffer.Stop()
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
func (docker *Docker) discover() []services.Endpoint {
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

	instances := make([]services.Endpoint, 0)

	for _, c := range containers {
		if c.State == "running" {
			serviceContainer := &services.Container{
				ID:      c.ID,
				Names:   c.Names,
				Image:   c.Image,
				Command: c.Command,
				State:   c.State,
				Labels:  c.Labels,
			}

			for _, port := range c.Ports {
				if port.PublicPort == 0 {
					log.WithFields(log.Fields{
						"containerName": serviceContainer.PrimaryName(),
						"privatePort":   port.PrivatePort,
					}).Debugf("Docker container does not expose port publically, not discovering")
					continue
				}

				id := serviceContainer.PrimaryName() + "-" + c.ID[:12] + "-" + strconv.Itoa(port.PrivatePort)

				endpoint := services.NewEndpointCore(id, "", time.Now(), observerType)
				endpoint.Host = "127.0.0.1"
				endpoint.PortType = services.PortType(port.Type)
				endpoint.Port = uint16(port.PublicPort)

				dims := map[string]string{
					"container_name":  serviceContainer.PrimaryName(),
					"container_image": c.Image,
				}

				orchestration := services.NewOrchestration("docker", services.DOCKER, dims, services.PUBLIC)

				si := &services.ContainerEndpoint{
					EndpointCore:  *endpoint,
					AltPort:       uint16(port.PrivatePort),
					Container:     *serviceContainer,
					Orchestration: *orchestration,
				}

				instances = append(instances, si)
			}
		}
	}

	return instances
}

// Shutdown the service differ routine
func (docker *Docker) Shutdown() {
	if docker.serviceDiffer != nil {
		docker.serviceDiffer.Stop()
	}
}
