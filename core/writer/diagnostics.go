package writer

import (
	"fmt"

	au "github.com/logrusorgru/aurora"
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
		au.Bold("Writer Status:\n").String()+
			"State:                    %s\n"+
			"DPs Sent:                 %d\n"+
			"Events Sent:              %d\n"+
			"DPs Buffered:             %d\n"+
			"Events Buffered:          %d\n"+
			"DPs Channel (len/cap) :   %d/%d\n"+
			"Events Channel (len/cap): %d/%d\n",
		au.Bold(sw.state.String()),
		au.Bold(sw.dpsSent),
		au.Bold(sw.eventsSent),
		au.Bold(len(sw.dpBuffer)),
		au.Bold(len(sw.eventBuffer)),
		au.Bold(len(sw.dpChan)),
		au.Bold(cap(sw.dpChan)),
		au.Bold(len(sw.eventChan)),
		au.Bold(cap(sw.eventChan)))
}

//func (sw *SignalFxWriter) InternalDatapoints() []*datapoint.Datapoint {
//}
