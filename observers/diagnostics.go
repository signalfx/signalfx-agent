package observers

import (
	"fmt"

	au "github.com/logrusorgru/aurora"
)

// DiagnosticText outputs human-readable text about the active observers.
func (om *ObserverManager) DiagnosticText() string {
	var out string
	out += au.Bold("Observers:\n").String()
	for i := range om.observers {
		out += fmt.Sprintf(
			" - %s\n",
			au.Bold(om.observers[i]._type))
	}
	return out
}
