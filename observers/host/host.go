// host observer that monitors the current host for active network listeners
// and reports them as service endpoints
// Use of this observer requires the CAP_SYS_PTRACE capability in Linux
package host

import (
	"fmt"
	"strconv"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/observers"
)

const (
	observerType = "host"
)

var logger = log.WithFields(log.Fields{"observerType": observerType})

// Observer that watches the current host
type Observer struct {
	serviceCallbacks *observers.ServiceCallbacks
	serviceDiffer    *observers.ServiceDiffer
	config           *Config
	hostInfoProvider hostInfoProvider
}

// Config specific to the host observer
type Config struct {
	config.ObserverConfig
	PollIntervalSeconds int `default:"10"`
}

func init() {
	observers.Register(observerType, func(cbs *observers.ServiceCallbacks) interface{} {
		return &Observer{
			serviceCallbacks: cbs,
			hostInfoProvider: &defaultHostInfoProvider{},
		}
	}, &Config{})
}

// Configure the host observer
func (o *Observer) Configure(config *Config) error {
	if o.serviceDiffer != nil {
		o.serviceDiffer.Stop()
	}

	o.serviceDiffer = &observers.ServiceDiffer{
		DiscoveryFn:     o.discover,
		IntervalSeconds: config.PollIntervalSeconds,
		Callbacks:       o.serviceCallbacks,
	}
	o.config = config

	o.serviceDiffer.Start()

	return nil
}

var portTypeMap = map[uint32]services.PortType{
	syscall.SOCK_STREAM: services.TCP,
	syscall.SOCK_DGRAM:  services.UDP,
}

func (o *Observer) discover() []services.Endpoint {
	conns, err := o.hostInfoProvider.AllConnectionStats()
	if err != nil {
		logger.WithError(err).Error("Could not get local network listeners")
		return nil
	}

	var endpoints []services.Endpoint
	for _, c := range conns {
		isIPSocket := c.Family == syscall.AF_INET || c.Family == syscall.AF_INET6
		isTCPOrUDP := c.Type == syscall.SOCK_STREAM || c.Type == syscall.SOCK_DGRAM
		isListening := c.Status == "LISTEN"

		// PID of 0 means that the listening file descriptor couldn't be mapped
		// back to a process's set of open file descriptors in /proc
		if !isIPSocket || !isTCPOrUDP || !isListening || c.Pid == 0 {
			continue
		}

		name, err := o.hostInfoProvider.ProcessNameFromPID(c.Pid)
		if err != nil {
			logger.WithFields(log.Fields{
				"pid":          c.Pid,
				"localAddress": c.Laddr.IP,
				"localPort":    c.Laddr.Port,
				"err":          err,
			}).Warn("Could not determine process name")
			continue
		}

		se := services.NewEndpointCore(
			fmt.Sprintf("%s-%d-%d", c.Laddr.IP, c.Laddr.Port, c.Pid), name, time.Now(), observerType)

		ip := c.Laddr.IP
		// An IP addr of 0.0.0.0 means it listens on all interfaces, including
		// localhost, so use that since we can't actually connect to 0.0.0.0.
		if ip == "0.0.0.0" {
			ip = "127.0.0.1"
		}

		se.Host = ip
		se.Port = uint16(c.Laddr.Port)
		se.PortType = portTypeMap[c.Type]

		se.AddDimension("pid", strconv.Itoa(int(c.Pid)))

		endpoints = append(endpoints, se)
	}
	return endpoints
}
