package docker

import (
	dtypes "github.com/docker/docker/api/types"
	"github.com/docker/go-connections/nat"
)

// FindHostMappedPort returns the port number of the docker port binding to the
// underlying host, or 0 if none exists.
func FindHostMappedPort(cont *dtypes.ContainerJSON, exposedPort nat.Port) int {
	bindings := cont.NetworkSettings.Ports[exposedPort]

	for i := range bindings {
		if port, err := nat.ParsePort(bindings[i].HostPort); err == nil {
			return port
		}
	}
	return 0
}
