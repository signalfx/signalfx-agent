package monitors

import (
	"github.com/signalfx/neo-agent/services"
)

const (
	// Collectd Monitor plugin name
	Collectd = "collectd"
)

// Monitor type
type Monitor interface {
	GetConfig(key string) (string, bool)
	Monitor(services services.ServiceInstances) error
	Start() error
	Stop() error
	Status() string
	String() string
}
