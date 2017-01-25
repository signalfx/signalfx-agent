package observers

import (
	"github.com/signalfx/neo-agent/services"
)

const (
	// Docker Observer plugin name
	Docker = "docker"
)

// Observer type
type Observer interface {
	Discover() (services.ServiceInstances, error)
	GetConfig(key string) (string, bool)
	String() string
}
