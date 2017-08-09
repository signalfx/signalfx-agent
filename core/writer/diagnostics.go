package writer

import (
	"fmt"

	. "github.com/logrusorgru/aurora"
)

func (s state) String() string {
	switch s {
	case stopped:
		return "stopped"
	case listening:
		return "listening"
	}
	return "unknown"
}

// DiagnosticText outputs a string that describes the state of the writer to a
// human.
func (sw *SignalFxWriter) DiagnosticText() string {
	return fmt.Sprintf(
		Bold("Writer Status:\n").String()+
			"State:                    %s\n"+
			"DPs Sent:                 %d\n"+
			"Events Sent:              %d\n"+
			"DPs Buffered:             %d\n"+
			"Events Buffered:          %d\n"+
			"DPs Channel (len/cap) :   %d/%d\n"+
			"Events Channel (len/cap): %d/%d\n",
		Bold(sw.state.String()),
		Bold(sw.dpsSent),
		Bold(sw.eventsSent),
		Bold(len(sw.dpBuffer)),
		Bold(len(sw.eventBuffer)),
		Bold(len(sw.dpChan)),
		Bold(cap(sw.dpChan)),
		Bold(len(sw.eventChan)),
		Bold(cap(sw.eventChan)))
}

//func (sw *SignalFxWriter) InternalDatapoints() []*datapoint.Datapoint {
//}
