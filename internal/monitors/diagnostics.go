package monitors

import (
	"fmt"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/core/services"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

func endpointToDiagnosticText(endpoint services.Endpoint, isMonitored bool) string {
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

	configurationText := "\n"
	for i := range mm.monitorConfigs {
		configurationText += fmt.Sprintf(
			"%s\n",
			utils.IndentLines(config.ToString(mm.monitorConfigs[i]), 2))
	}

	activeMonText := ""
	for i := range mm.activeMonitors {
		am := mm.activeMonitors[i]

		serviceStats := ""
		if am.endpoint != nil {
			serviceStats = fmt.Sprintf(
				"Discovery Rule: %s\n"+
					"Monitored Endpoint ID: %s\n",
				am.config.MonitorConfigCore().DiscoveryRule,
				am.endpoint.Core().ID)
		}
		activeMonText += fmt.Sprintf(
			"%2d. %s\n"+
				"    Reporting Interval (seconds): %d\n"+
				"%s"+
				"    Config:\n%s\n",
			i+1, am.config.MonitorConfigCore().Type,
			am.config.MonitorConfigCore().IntervalSeconds,
			utils.IndentLines(serviceStats, 4),
			utils.IndentLines(config.ToString(am.config), 6))
		i++
	}

	var discoveredEndpointsText string
	for _, endpoint := range mm.discoveredEndpoints {
		discoveredEndpointsText += endpointToDiagnosticText(endpoint, mm.isEndpointMonitored(endpoint))
	}
	if len(discoveredEndpointsText) == 0 {
		discoveredEndpointsText = "None\n"
	}

	return fmt.Sprintf(
		"Monitor Configurations (Not necessarily active):\n"+
			"%s"+
			"Active Monitors:\n"+
			"%s"+
			"Discovered Endpoints:\n"+
			"%s\n"+
			"Bad Monitor Configurations:\n\n"+
			"%s\n",
		configurationText,
		activeMonText,
		discoveredEndpointsText,
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

func badConfigText(confs map[uint64]*config.MonitorConfig) string {
	if len(confs) > 0 {
		var text string
		for k := range confs {
			conf := confs[k]
			text += fmt.Sprintf("Type: %s\nError: %s\n\n",
				conf.Type, conf.ValidationError)
		}
		return text
	}
	return "None"
}
