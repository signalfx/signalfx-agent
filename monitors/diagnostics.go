package monitors

import (
	"fmt"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/utils"
)

func serviceToDiagnosticText(service services.Endpoint, isMonitored bool) string {
	var containerInfo string
	for k, v := range services.EndpointAsMap(service) {
		containerInfo += fmt.Sprintf("%s: %s\n", k, v)
	}
	var unmonitoredText string
	if !isMonitored {
		unmonitoredText = "(Unmonitored)"
	}

	text := fmt.Sprintf(
		"- Internal ID: %s %s\n"+
			"%s\n",
		service.ID(),
		unmonitoredText,
		utils.IndentLines(containerInfo, 2))
	return text

}

// DiagnosticText returns a string to be served on the diagnostic socket
func (mm *MonitorManager) DiagnosticText() string {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	activeMonText := ""
	for i := range mm.activeMonitors {
		am := mm.activeMonitors[i]

		serviceStats := "\n"
		if len(am.serviceSet) > 0 {
			serviceText := ""
			for id := range am.serviceSet {
				serviceText += serviceToDiagnosticText(am.serviceSet[id], true)
			}
			serviceStats = fmt.Sprintf(
				"Discovery Rule: %s\nServices:\n%s",
				am.config.CoreConfig().DiscoveryRule, serviceText)
		}
		activeMonText += fmt.Sprintf(
			"%2d. %s\n"+
				"    Reporting Interval (seconds): %d\n"+
				"%s\n",
			i+1, am.config.CoreConfig().Type,
			am.config.CoreConfig().IntervalSeconds,
			utils.IndentLines(serviceStats, 4))
		i++
	}

	var discoveredServicesText string
	for _, service := range mm.discoveredServices {
		discoveredServicesText += serviceToDiagnosticText(service, mm.isServiceMonitored(service))
	}
	if len(discoveredServicesText) == 0 {
		discoveredServicesText = "None\n"
	}

	return fmt.Sprintf(
		"Active Monitors:\n"+
			"%s"+
			"Discovered Endpoints:\n"+
			"%s\n"+
			"Bad Monitor Configurations:\n"+
			"%s\n",
		activeMonText,
		discoveredServicesText,
		badConfigText(mm.badConfigs))
}

// InternalMetrics returns a list of datapoints about the internal status of
// the monitors
func (mm *MonitorManager) InternalMetrics() []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		sfxclient.Gauge("sfxagent.active_monitors", nil, int64(len(mm.activeMonitors))),
		sfxclient.Gauge("sfxagent.configured_monitors", nil, int64(len(mm.monitorConfigs))),
		sfxclient.Gauge("sfxagent.discovered_endpoints", nil, int64(len(mm.discoveredServices))),
	}
}

func badConfigText(confs []config.MonitorCustomConfig) string {
	if len(confs) > 0 {
		var text string
		for i := range confs {
			conf := confs[i].CoreConfig()
			text += fmt.Sprintf("Type: %s\nError: %s\n\n",
				conf.Type, conf.ValidationError)
		}
		return text
	}
	return "None"
}
