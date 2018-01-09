// +build ignore

package neopy

import (
	"bytes"
	"text/template"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/monitors"
	log "github.com/sirupsen/logrus"
)

// DDMonitorTypePrefix is the prefix for all DataDog monitor type strings
const DDMonitorTypePrefix = "dd/"

// DDConfig is a generic config struct that allows configuration of DataDog
// checks.  DD checks accept three types of config: agent config (global agent
// config), init config (check-specific but applied across all instances of the
// monitored service), and instance config (specifying the individual instances
// that are being monitored).
type DDConfig struct {
	config.MonitorConfig
	// These next three correspond to the three parameters for the __init__
	// method of DD Check classes.
	AgentConfig       map[string]interface{}   `json:"agentConfig"`
	InitConfig        map[string]interface{}   `yaml:"init_config" default:"{}" json:"init_config"`
	InstancesOverride []map[string]interface{} `yaml:"instances" default:"[]"`

	AgentConfigOverride map[string]interface{}   `yaml:"agentConfig" default:"{}"`
	InstanceTemplate    map[string]string        `yaml:"instanceTemplate" default:"{}"`
	FinalInstances      []map[string]interface{} `json:"instances"`
}

// DDCheck represents a Datadog check that may or may not be configured via
// service discovery.
type DDCheck struct {
	id         monitors.MonitorID
	serviceSet map[services.ID]services.Endpoint
	DPs        chan<- *datapoint.Datapoint
	config     *DDConfig
}

// Configure the DD check.  This will update the python runner with the
// configured services.
func (ddc *DDCheck) Configure(config *DDConfig) bool {
	ddc.config = config

	ddc.config.AgentConfig = config.AgentConfigOverride
	ddc.config.AgentConfig["procfs_path"] = config.ProcFSPath
	ddc.config.AgentConfig["version"] = "0.0.0-signalfx"

	Instance().sendDatapointsForMonitorTo(config.ID, ddc.DPs)

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
		return Instance().ConfigureInPython(config)
	}
	return true
}

// AddService adds a service endpoint to the Python runner.  This is only
// called by the monitor manager.
func (ddc *DDCheck) AddService(service services.Endpoint) {
	ddc.serviceSet[service.ID()] = service
	// We just reconfigured the whole python instance, which will ultimately
	// destroy and recreate the check since DD checks have no
	// hot-reloading capabilities
	ddc.Configure(ddc.config)
}

// RemoveService removes a service endpoint from Python.  This is only called
// by the monitor manager.
func (ddc *DDCheck) RemoveService(service services.Endpoint) {
	delete(ddc.serviceSet, service.ID())
	ddc.Configure(ddc.config)
}

// This creates the "instances" config for the DD check.  It renders the
// templates in `InstanceTemplate` using go templates.
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

func (ddc *DDCheck) renderInstanceTemplateValue(t string, service services.Endpoint) (string, error) {
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

func registerDDCheck(_type string) {
	monitors.Register(_type, func(id monitors.MonitorID) interface{} {
		return &DDCheck{
			serviceSet: make(map[services.ID]services.Endpoint),
			id:         id,
		}
	}, &DDConfig{})
}

// Shutdown the instance both in neo-agent and python
func (ddc *DDCheck) Shutdown() {
	Instance().ShutdownMonitor(ddc.config.ID)
}
