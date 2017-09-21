package monitors

import (
	"fmt"

	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/utils"

	au "github.com/logrusorgru/aurora"
)

func serviceToDiagnosticText(service services.Endpoint, isMonitored bool) string {
	var containerInfo string
	for k, v := range services.EndpointAsMap(service) {
		containerInfo += fmt.Sprintf("%s: %s\n", k, au.Bold(v))
	}
	var unmonitoredText string
	if !isMonitored {
		unmonitoredText = au.Red("(Unmonitored)").String()
	}

	text := fmt.Sprintf(
		au.Bold("- ").String()+"Internal ID: %s %s\n"+
			"%s\n",
		au.Bold(service.ID()),
		au.Bold(unmonitoredText),
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

		serviceStats := "Service Endpoints: None\n"
		if len(am.serviceSet) > 0 {
			serviceText := ""
			for id := range am.serviceSet {
				serviceText += serviceToDiagnosticText(am.serviceSet[id], true)
			}
			serviceStats = fmt.Sprintf(
				"Discovery Rule: %s\nServices:\n%s",
				au.Bold(am.config.CoreConfig().DiscoveryRule), serviceText)
		}
		activeMonText += fmt.Sprintf(
			"%2d. %s\n"+
				"    Reporting Interval (seconds): %d\n"+
				"%s\n",
			i+1, au.Bold(am.config.CoreConfig().Type),
			au.Bold(am.config.CoreConfig().IntervalSeconds),
			utils.IndentLines(serviceStats, 4))
		i++
	}

	var discoveredServicesText string
	for _, service := range mm.discoveredServices {
		discoveredServicesText += serviceToDiagnosticText(service, mm.isServiceMonitored(service))
	}
	if len(discoveredServicesText) == 0 {
		discoveredServicesText = "None"
	}

	return fmt.Sprintf(
		au.Bold("Active Monitors:\n").String()+
			"%s"+
			au.Bold("Discovered Endpoints:\n").String()+
			"%s\n",
		activeMonText,
		discoveredServicesText)
}
