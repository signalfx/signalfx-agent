// This package holds datadog-specific logic that helps configure and adapt
// their plugins to our system.
package neopy

import (
	"bytes"
	"text/template"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/observers"
	log "github.com/sirupsen/logrus"
)

const DDMonitorTypePrefix = "dd/"

type DDConfig struct {
	config.MonitorConfig
	// These next three correspond to the three parameters for the __init__
	// method of DD Check classes.
	InitConfig          map[string]interface{} `mapstructure:"init_config" default:"{}" json:"init_config"`
	AgentConfig         map[string]interface{} `json:"agentConfig"`
	AgentConfigOverride map[string]interface{} `mapstructure:"agentConfig" default:"{}"`

	InstancesOverride []map[string]interface{} `mapstructure:"instances" default:"[]"`
	InstanceTemplate  map[string]string        `default:"{}"`
	FinalInstances    []map[string]interface{} `json:"instances"`
}

// Represents a datadog check that may or may not be configured via service
// discovery.
type DDCheck struct {
	serviceSet map[observers.ServiceID]*observers.ServiceInstance
	DPs        chan<- *datapoint.Datapoint
	config     *DDConfig
}

func (ddc *DDCheck) Configure(config *DDConfig) bool {
	ddc.config = config

	ddc.config.AgentConfig = config.AgentConfigOverride
	ddc.config.AgentConfig["procfs_path"] = config.ProcFSPath
	ddc.config.AgentConfig["version"] = "0.0.0-signalfx"

	GetInstance().SendDatapointsForMonitorTo(config.Id, ddc.DPs)

	if len(config.InstancesOverride) == 0 && config.DiscoveryRule == "" {
		log.WithFields(log.Fields{
			"monitorType": config.Type,
		}).Error("DataDog plugin must have at least one instance or a discovery rule in order to be useful")
		return false
	}

	if config.DiscoveryRule != "" && len(config.InstanceTemplate) == 0 {
		log.WithFields(log.Fields{
			"monitorType":   config.Type,
			"discoveryRule": config.DiscoveryRule,
		}).Error("DataDog plugin with discovery rule must specify an instanceTemplate to map services")
		return false
	}

	config.FinalInstances = append(ddc.instancesFromServices(), config.InstancesOverride...)

	if len(config.FinalInstances) > 0 {
		return GetInstance().ConfigureInPython(config)
	}
	return true
}

// TODO: Allow dynamic configuration of "instances" upon service discovery
func (ddc *DDCheck) AddService(service *observers.ServiceInstance) {
	ddc.serviceSet[service.ID] = service
	ddc.Configure(ddc.config)
}

func (ddc *DDCheck) RemoveService(service *observers.ServiceInstance) {
	delete(ddc.serviceSet, service.ID)
	ddc.Configure(ddc.config)
}

func (ddc *DDCheck) instancesFromServices() []map[string]interface{} {
	instances := make([]map[string]interface{}, 0)
	for _, service := range ddc.serviceSet {
		inst := make(map[string]interface{})
		for k, v := range ddc.config.InstanceTemplate {
			val, err := ddc.renderInstanceTemplateValue(v, service)
			if err != nil {
				log.WithFields(log.Fields{
					"instanceTemplateValue": v,
					"service":               service,
					"error":                 err,
				}).Error("Could not render DataDog instance template for service")
				return nil
			}
			inst[k] = val
		}
		instances = append(instances, inst)
	}
	return instances
}

func (ddc *DDCheck) renderInstanceTemplateValue(t string, service *observers.ServiceInstance) (string, error) {
	tmpl, err := template.New("instanceValue").Parse(t)
	if err != nil {
		return "", err
	}

	output := bytes.Buffer{}
	err = tmpl.Execute(&output, service)
	if err != nil {
		return "", err
	}

	return output.String(), nil
}

func RegisterDDCheck(_type string) {
	monitors.Register(_type, func() interface{} {
		return &DDCheck{
			serviceSet: make(map[observers.ServiceID]*observers.ServiceInstance),
		}
	}, &DDConfig{})
}

func (dc *DDCheck) Shutdown() {
	GetInstance().ShutdownMonitor(dc.config.Id)
}
