package writer

import (
	"fmt"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
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

func (sw *SignalFxWriter) averageDPM() uint {
	minutesActive := time.Since(sw.startTime).Minutes()
	return uint(float64(sw.dpsSent) / minutesActive)
}

// DiagnosticText outputs a string that describes the state of the writer to a
// human.
func (sw *SignalFxWriter) DiagnosticText() string {
	return fmt.Sprintf(
		"Writer Status:\n"+
			"Global Dims:              %s\n"+
			"Host ID Dims:             %s\n"+
			"State:                    %s\n"+
			"Average DPM:              %d\n"+
			"DPs Sent:                 %d\n"+
			"Events Sent:              %d\n"+
			"DPs Buffered:             %d\n"+
			"Events Buffered:          %d\n"+
			"DPs Channel (len/cap) :   %d/%d\n"+
			"Events Channel (len/cap): %d/%d\n",
		sw.conf.GlobalDimensions,
		sw.hostIDDims,
		sw.state.String(),
		sw.averageDPM(),
		sw.dpsSent,
		sw.eventsSent,
		len(sw.dpBuffer),
		len(sw.eventBuffer),
		len(sw.dpChan),
		cap(sw.dpChan),
		len(sw.eventChan),
		cap(sw.eventChan))
}

// InternalMetrics returns a set of metrics showing how the writer is currently
// doing.
func (sw *SignalFxWriter) InternalMetrics() []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		sfxclient.Cumulative("sfxagent.datapoints_sent", nil, int64(sw.dpsSent)),
		sfxclient.Cumulative("sfxagent.events_sent", nil, int64(sw.eventsSent)),
		sfxclient.Gauge("sfxagent.datapoints_buffered", nil, int64(len(sw.dpBuffer))),
		sfxclient.Gauge("sfxagent.events_buffered", nil, int64(len(sw.eventBuffer))),
	}
}
