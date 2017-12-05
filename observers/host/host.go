// host observer that monitors the current host for active network listeners
// and reports them as service endpoints
package host

import (
	"github.com/shirou/gopsutil/net"
	log "github.com/sirupsen/logrus"

	"github.com/docker/engine-api/client"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/observers"
)

const (
	observerType = "host"
)

var logger = log.WithFields(log.Fields{"observerType": observerType})

// Docker observer plugin
type Observer struct {
	client           *client.Client
	serviceCallbacks *observers.ServiceCallbacks
	serviceDiffer    *observers.ServiceDiffer
	config           *Config
}

// Config specific to the Docker observer
type Config struct {
	config.ObserverConfig
}

func init() {
	observers.Register(observerType, func(cbs *observers.ServiceCallbacks) interface{} {
		return &Observer{
			serviceCallbacks: cbs,
		}
	}, &Config{})
}

// Configure the docker client
func (o *Observer) Configure(config *Config) bool {
	if docker.serviceDiffer != nil {
		docker.serviceDiffer.Stop()
	}

	o.serviceDiffer = &observers.ServiceDiffer{
		DiscoveryFn:     o.discover,
		IntervalSeconds: config.PollIntervalSeconds,
		Callbacks:       o.serviceCallbacks,
	}
	o.config = config

	o.serviceDiffer.Start()

	return true
}

func (o *Observer) discover() {
	conns, err := net.Connections("all")

}
