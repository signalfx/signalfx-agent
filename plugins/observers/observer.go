package observers

import "github.com/signalfx/neo-agent/services"

const (
	// Docker Observer plugin name
	Docker = "docker"
	// Kubernetes Observer plugin name
	Kubernetes = "kubernetes"
)

// Observer type
type Observer interface {
	Discover() (services.ServiceInstances, error)
	String() string
}
