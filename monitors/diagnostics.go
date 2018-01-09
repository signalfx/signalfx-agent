package monitors

import (
	"fmt"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/utils"
)

func serviceToDiagnosticText(endpoint services.Endpoint, isMonitored bool) string {
	var containerInfo string
	endpointMap := services.EndpointAsMap(endpoint)
	sortedKeys := utils.SortMapKeys(endpointMap)
	for _, k := range sortedKeys {
		val := endpointMap[k]
		containerInfo += fmt.Sprintf("%s: %v\n", k, val)
	}
	var unmonitoredText string
	if !isMonitored {
		unmonitoredText = "(Unmonitored)"
	}

	text := fmt.Sprintf(
		"- Internal ID: %s %s\n"+
			"%s\n",
		endpoint.Core().ID,
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
		if am.endpoint != nil {
			serviceText := serviceToDiagnosticText(am.endpoint, true)
			serviceStats = fmt.Sprintf(
				"Discovery Rule: %s\nService:\n%s",
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
	for _, endpoint := range mm.discoveredEndpoints {
		discoveredServicesText += serviceToDiagnosticText(endpoint, mm.isEndpointMonitored(endpoint))
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
		sfxclient.Gauge("sfxagent.discovered_endpoints", nil, int64(len(mm.discoveredEndpoints))),
	}
}

func badConfigText(confs []*config.MonitorConfig) string {
	if len(confs) > 0 {
		var text string
		for i := range confs {
			conf := confs[i]
			text += fmt.Sprintf("Type: %s\nError: %s\n\n",
				conf.Type, conf.ValidationError)
		}
		return text
	}
	return "None"
}
