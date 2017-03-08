package observers

import "github.com/signalfx/neo-agent/services"

const (
	// Docker Observer plugin name
	Docker = "docker"
	// Kubernetes Observer plugin name
	Kubernetes = "kubernetes"
	// Mesosphere Observer plugin name
	Mesosphere = "mesosphere"
)

// Observer type
type Observer interface {
	Discover() (services.Instances, error)
	String() string
}
