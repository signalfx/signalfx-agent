// Package host observer that monitors the current host for active network
// listeners and reports them as service endpoints Use of this observer
// requires the CAP_SYS_PTRACE and CAP_DAC_READ_SEARCH capability in Linux.
package host

import (
	"fmt"
	"strconv"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/core/services"
	"github.com/signalfx/signalfx-agent/internal/observers"
)

const (
	observerType = "host"
)

// OBSERVER(host): Looks at the current host for listening network endpoints.
// It uses the `/proc` filesystem and requires the `SYS_PTRACE` and
// `DAC_READ_SEARCH` capabilities so that it can determine what processes own
// the listening sockets.
//
// It will look for all listening sockets on TCP and UDP over IPv4 and IPv6.

// DIMENSION(pid): The PID of the process that owns the listening endpoint

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
	PollIntervalSeconds int `default:"10" yaml:"pollIntervalSeconds"`
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
		// TODO: Add support for ipv6 to all observers
		isIPSocket := c.Family == syscall.AF_INET
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

		dims := map[string]string{
			"pid": strconv.Itoa(int(c.Pid)),
		}

		se := services.NewEndpointCore(
			fmt.Sprintf("%s-%d-%d", c.Laddr.IP, c.Laddr.Port, c.Pid), name, observerType, dims)

		ip := c.Laddr.IP
		// An IP addr of 0.0.0.0 means it listens on all interfaces, including
		// localhost, so use that since we can't actually connect to 0.0.0.0.
		if ip == "0.0.0.0" {
			ip = "127.0.0.1"
		}

		se.Host = ip
		se.Port = uint16(c.Laddr.Port)
		se.PortType = portTypeMap[c.Type]

		endpoints = append(endpoints, se)
	}
	return endpoints
}
