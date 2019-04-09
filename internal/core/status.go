package core

import (
	"fmt"
	"net/http"
	"time"

	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors/kubernetes/leadership"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

func (a *Agent) diagnosticTextHandler(rw http.ResponseWriter, req *http.Request) {
	section := req.URL.Query().Get("section")
	rw.Write([]byte(a.DiagnosticText(section)))
}

var startTime time.Time

func init() {
	startTime = time.Now()
}

// DiagnosticText returns a simple textual output of the agent's status
func (a *Agent) DiagnosticText(section string) string {
	var out string
	if section == "" || section == "all" {
		uptime := time.Now().Sub(startTime).Round(1 * time.Second).String()
		out +=
			"SignalFx Agent version:           " + constants.Version + "\n" +
				"Agent uptime:                     " + uptime + "\n" +
				a.observers.DiagnosticText() + "\n" +
				a.monitors.SummaryDiagnosticText() + "\n" +
				a.writer.DiagnosticText() + "\n"

		k8sLeader := leadership.CurrentLeader()
		if k8sLeader != "" {
			out += fmt.Sprintf("Kubernetes Leader Node:           %s\n", k8sLeader)
		}

		if section == "" {
			out += "\n" + utils.StripIndent(`
			  Additional status commands:

			  signalfx-agent status config - show resolved config in use by agent
			  signalfx-agent status endpoints - show discovered endpoints
			  signalfx-agent status monitors - show active monitors
			  signalfx-agent status all - show everything
			  `)
		}
	}

	if section == "config" || section == "all" {
		out += "Agent Configuration:\n" +
			utils.IndentLines(config.ToString(a.lastConfig), 2) + "\n"
	}

	if section == "monitors" || section == "all" {
		out += a.monitors.DiagnosticText() + "\n"
	}

	if section == "endpoints" || section == "all" {
		out += a.monitors.EndpointsDiagnosticText()
	}

	return out
}
