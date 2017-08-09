package monitors

import (
	"fmt"
	"strings"

	"github.com/signalfx/neo-agent/observers"
	"github.com/signalfx/neo-agent/utils"

	. "github.com/logrusorgru/aurora"
)

func serviceToDiagnosticText(service *observers.ServiceInstance, isMonitored bool) string {
	var containerInfo string
	if service.Container != nil {
		containerInfo = "Container Name (ID): "
		containerInfo += strings.Join(service.Container.Names, "; ")
		containerInfo += " (" + Bold(service.Container.ID[:12]).String() + ")\n"
		containerInfo += "Image: " + Bold(service.Container.Image).String() + "\n"
	}
	var unmonitoredText string
	if !isMonitored {
		unmonitoredText = "(Unmonitored)"
	}

	text := fmt.Sprintf(
		"- Internal ID: %s %s\n"+
			"%s",
		Bold(service.ID),
		Bold(unmonitoredText),
		utils.IndentLines(containerInfo, 2))
	if isMonitored {
		return text
	} else {
		return Red(text).String()
	}

}

// DiagnosticText returns a string to be served on the diagnostic socket
func (mm *MonitorManager) DiagnosticText() string {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	activeMonText := ""
	for i := range mm.activeMonitors {
		am := mm.activeMonitors[i]

		var serviceStats string
		if am.config.DiscoveryRule != "" {
			serviceText := ""
			for id := range am.serviceSet {
				serviceText += serviceToDiagnosticText(mm.discoveredServices[id], true)
			}
			serviceStats = utils.IndentLines(fmt.Sprintf(
				"Discovery Rule: %s\nServices:\n%s",
				Bold(am.config.DiscoveryRule), serviceText), 4)
		}
		activeMonText += fmt.Sprintf(
			"%2d. %s\n"+
				"    Reporting Interval (seconds): %d\n"+
				"%s\n",
			i+1, Bold(am.config.Type),
			Bold(am.config.IntervalSeconds),
			serviceStats)
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
		Bold("Active Monitors:\n").String()+
			"%s"+
			Bold("Discovered Services:\n").String()+
			"%s\n",
		activeMonText,
		discoveredServicesText)
}
