// Package docker is an observer that watches a docker daemon and reports
// container ports as service endpoints.
package docker

import (
	"context"
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	dtypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	dockercommon "github.com/signalfx/signalfx-agent/internal/core/common/docker"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/core/services"
	"github.com/signalfx/signalfx-agent/internal/observers"
)

const (
	observerType     = "docker"
	dockerAPIVersion = "v1.22"
)

// OBSERVER(docker): Queries the Docker Engine API for running containers.  If
// you are using Kubernetes, you should use the [k8s-api
// observer](./k8s-api.md) instead of this.
//
// Note that you will need permissions to access the Docker engine API.  For a
// Docker domain socket URL, this means that the agent needs to have read
// permissions on the socket.  We don't currently support authentication for
// HTTP URLs.

// ENDPOINT_TYPE(ContainerEndpoint): true

var logger = log.WithFields(log.Fields{"observerType": observerType})

// Docker observer plugin
type Docker struct {
	serviceCallbacks *observers.ServiceCallbacks
	config           *Config
	cancel           func()

	endpointsByContainerID map[string][]services.Endpoint
}

// Config specific to the Docker observer
type Config struct {
	config.ObserverConfig
	DockerURL string `yaml:"dockerURL" default:"unix:///var/run/docker.sock"`
	// A mapping of container label names to dimension names that will get
	// applied to the metrics of all discovered services. The corresponding
	// label values will become the dimension value for the mapped name.  E.g.
	// `io.kubernetes.container.name: container_spec_name` would result in a
	// dimension called `container_spec_name` that has the value of the
	// `io.kubernetes.container.name` container label.
	LabelsToDimensions map[string]string `yaml:"labelsToDimensions"`
	// If true, the "Config.Hostname" field (if present) of the docker
	// container will be used as the discovered host that is used to configure
	// monitors.  If false or if no hostname is configured, the field
	// `NetworkSettings.IPAddress` is used instead.
	UseHostnameIfPresent bool `yaml:"useHostnameIfPresent"`
}

func init() {
	observers.Register(observerType, func(cbs *observers.ServiceCallbacks) interface{} {
		return &Docker{
			serviceCallbacks:       cbs,
			endpointsByContainerID: make(map[string][]services.Endpoint),
		}
	}, &Config{})
}

// Configure the docker client
func (docker *Docker) Configure(config *Config) error {
	defaultHeaders := map[string]string{"User-Agent": "signalfx-agent"}

	client, err := client.NewClient(config.DockerURL, dockerAPIVersion, nil, defaultHeaders)
	if err != nil {
		return errors.Wrapf(err, "Could not create docker client")
	}

	docker.config = config

	var ctx context.Context
	ctx, docker.cancel = context.WithCancel(context.Background())

	err = dockercommon.ListAndWatchContainers(ctx, client, docker.changeHandler, nil, logger)
	if err != nil {
		logger.WithError(err).Error("Could not list docker containers")
		return err
	}
	return nil
}

func (docker *Docker) changeHandler(old *dtypes.ContainerJSON, new *dtypes.ContainerJSON) {
	var newEndpoints []services.Endpoint
	var oldEndpoints []services.Endpoint

	if old != nil {
		oldEndpoints = docker.endpointsByContainerID[old.ID]
		delete(docker.endpointsByContainerID, old.ID)
	}

	if new != nil {
		newEndpoints = docker.endpointsForContainer(new)
		docker.endpointsByContainerID[new.ID] = newEndpoints
	}

	// Prevent spurious churn of endpoints if they haven't changed
	if reflect.DeepEqual(newEndpoints, oldEndpoints) {
		return
	}

	// If it is an update, there will be a remove and immediately subsequent
	// add.
	for i := range oldEndpoints {
		log.Debugf("Removing Docker endpoint from container %s", old.ID)
		docker.serviceCallbacks.Removed(oldEndpoints[i])
	}

	for i := range newEndpoints {
		log.Debugf("Adding Docker endpoint for container %s", new.ID)
		docker.serviceCallbacks.Added(newEndpoints[i])
	}
}

// Discover services by querying docker api
func (docker *Docker) endpointsForContainer(cont *dtypes.ContainerJSON) []services.Endpoint {
	instances := make([]services.Endpoint, 0)

	if cont.State.Running && !cont.State.Paused {
		serviceContainer := &services.Container{
			ID:      cont.ID,
			Names:   []string{cont.Name},
			Image:   cont.Config.Image,
			Command: strings.Join(cont.Config.Cmd, " "),
			State:   cont.State.Status,
			Labels:  cont.Config.Labels,
		}

		for portObj := range cont.Config.ExposedPorts {
			port := portObj.Int()
			protocol := portObj.Proto()

			id := serviceContainer.PrimaryName() + "-" + cont.ID[:12] + "-" + strconv.Itoa(int(port))

			endpoint := services.NewEndpointCore(id, "", observerType)
			if docker.config.UseHostnameIfPresent && cont.Config.Hostname != "" {
				endpoint.Host = cont.Config.Hostname
			} else {
				// Use the IP Address of the first network we iterate over.
				// This can be made configurable if so desired.
				for _, n := range cont.NetworkSettings.Networks {
					endpoint.Host = n.IPAddress
					break
				}
			}
			endpoint.PortType = services.PortType(strings.ToUpper(protocol))
			endpoint.Port = uint16(port)

			orchDims := map[string]string{}
			for k, dimName := range docker.config.LabelsToDimensions {
				if v := cont.Config.Labels[k]; v != "" {
					orchDims[dimName] = v
				}
			}

			orchestration := services.NewOrchestration("docker", services.DOCKER, orchDims, services.PRIVATE)

			si := &services.ContainerEndpoint{
				EndpointCore:  *endpoint,
				AltPort:       uint16(dockercommon.FindHostMappedPort(cont, portObj)),
				Container:     *serviceContainer,
				Orchestration: *orchestration,
			}

			instances = append(instances, si)
		}
	}

	return instances
}

// Shutdown the service differ routine
func (docker *Docker) Shutdown() {
	if docker.cancel != nil {
		docker.cancel()
	}
}
