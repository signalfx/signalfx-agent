package observers

import (
	"fmt"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
)

// DiagnosticText outputs human-readable text about the active observers.
func (om *ObserverManager) DiagnosticText() string {
	var out string
	out += "Observers:\n"
	for i := range om.observers {
		out += fmt.Sprintf(
			" - %s\n",
			om.observers[i]._type)
	}
	return out
}

// InternalMetrics returns a list of datapoints relevant to the internal status
// of Observers
func (om *ObserverManager) InternalMetrics() []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		sfxclient.Gauge("sfxagent.active_observers", nil, int64(len(om.observers))),
	}
}
