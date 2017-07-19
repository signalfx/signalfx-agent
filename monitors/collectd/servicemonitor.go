package collectd

import (
	"text/template"

	"github.com/signalfx/neo-agent/observers"
	log "github.com/sirupsen/logrus"
)

// This is intended to be embedded in the individual monitors that represent
// dynamically configured (i.e. from service discovery) collectd plugins
type ServiceMonitorCore struct {
	*BaseMonitor
	ServiceSet map[observers.ServiceID]*observers.ServiceInstance
}

func NewServiceMonitorCore(template *template.Template) *ServiceMonitorCore {
	return &ServiceMonitorCore{
		ServiceSet:  make(map[observers.ServiceID]*observers.ServiceInstance),
		BaseMonitor: NewBaseMonitor(template),
	}
}

func (smc *ServiceMonitorCore) AddService(service *observers.ServiceInstance) {
	log.WithFields(log.Fields{
		"service":             *service,
		"monitorTemplateName": smc.Template.Name(),
	}).Debug("Collectd monitor got service")

	smc.ServiceSet[service.ID] = service

	smc.Context.SetServices(append(smc.Context.GetServices(), service))

	smc.setOrchestrationDimensions()
	smc.WriteConfigForPluginAndRestart()
}

func (smc *ServiceMonitorCore) RemoveService(service *observers.ServiceInstance) {
	delete(smc.ServiceSet, service.ID)

	services := smc.Context.GetServices()
	for i := range services {
		if services[i].ID == service.ID {
			smc.Context.SetServices(append(services[:i], services[i+1:]...))
			break
		}
	}

	smc.setOrchestrationDimensions()
	smc.WriteConfigForPluginAndRestart()
}

// Add dimensions from the service orchestrations
func (smc *ServiceMonitorCore) setOrchestrationDimensions() {
	dims := smc.Context.GetDimensions()

	for _, service := range smc.Context.GetServices() {
		if service.Orchestration == nil {
			continue
		}

		for k, v := range service.Orchestration.Dims {
			dims[k] = v
		}
	}
	smc.Context.SetDimensions(dims)
}

func (smc *ServiceMonitorCore) ServicesInSlice() []*observers.ServiceInstance {
	services := make([]*observers.ServiceInstance, len(smc.ServiceSet))
	for _, s := range smc.ServiceSet {
		services = append(services, s)
	}
	return services
}
