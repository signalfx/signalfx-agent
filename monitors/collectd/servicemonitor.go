package collectd

import (
	"text/template"

	"github.com/davecgh/go-spew/spew"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	log "github.com/sirupsen/logrus"
)

// ServiceMonitorCore is intended to be embedded in the individual monitors
// that represent dynamically configured (i.e. from service discovery) collectd
// plugins.  It has most of the logic those type of monitors will need to
// render configuration upon service changes and manage restarting collectd.
type ServiceMonitorCore struct {
	*BaseMonitor
	ServiceSet map[services.ID]services.Endpoint
}

// NewServiceMonitorCore creates a new instance with no configuration
func NewServiceMonitorCore(template *template.Template) *ServiceMonitorCore {
	return &ServiceMonitorCore{
		ServiceSet:  make(map[services.ID]services.Endpoint),
		BaseMonitor: NewBaseMonitor(template),
	}
}

// SetConfigurationAndRun sets the configuration to be used when rendering
// templates, and writes config before queueing a collectd restart.
func (smc *ServiceMonitorCore) SetConfigurationAndRun(conf *config.MonitorConfig, commonEndpointConfig interface{}) bool {
	if commonEndpointConfig != nil {
		if ok := smc.Context.InjectConfigStruct(commonEndpointConfig); !ok {
			return false
		}
	}

	if !smc.BaseMonitor.SetConfiguration(conf) {
		return false
	}

	if len(smc.ServiceSet) > 0 {
		return smc.WriteConfigForPluginAndRestart()
	}
	return true
}

// AddService adds a service to the monitor, rerenders the collectd conf for
// the monitor and queus a collectd restart.
func (smc *ServiceMonitorCore) AddService(service services.Endpoint) {

	smc.ServiceSet[service.ID()] = service

	smc.Context.SetEndpointInstances(append([]services.Endpoint{service}, smc.Context.EndpointInstances()...))

	log.WithFields(log.Fields{
		"serviceEndpoint":     service,
		"monitorTemplateName": smc.Template.Name(),
		"allEndpoints":        spew.Sdump(smc.Context.EndpointInstances()),
	}).Debug("Collectd monitor got service")

	smc.WriteConfigForPluginAndRestart()
}

// RemoveService removes the service for the monitor and rerenders collectd
// config and queues a collectd restart.
func (smc *ServiceMonitorCore) RemoveService(service services.Endpoint) {
	delete(smc.ServiceSet, service.ID())

	services := smc.Context.EndpointInstances()
	for i := range services {
		if services[i].ID() == service.ID() {
			smc.Context.SetEndpointInstances(append(services[:i], services[i+1:]...))
			break
		}
	}

	smc.WriteConfigForPluginAndRestart()
}
