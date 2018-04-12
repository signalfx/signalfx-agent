package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	// Import for side-effect of registering http handler
	_ "net/http/pprof"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/network"
	log "github.com/sirupsen/logrus"
)

// VersionLine should be populated by the startup logic to contain version
// information that can be reported in diagnostics.
var VersionLine string

// Serves the diagnostic status on the domain socket
func (a *Agent) serveDiagnosticInfo(socketPath string) error {
	if a.diagnosticSocketStop != nil {
		a.diagnosticSocketStop()
	}

	if err := os.MkdirAll(filepath.Dir(socketPath), 0700); err != nil {
		return err
	}

	var err error
	a.diagnosticSocketStop, err = network.RunSimpleSocketServer(socketPath, func(_ net.Conn) string {
		return a.DiagnosticText()
	}, func(err error) {
		log.WithFields(log.Fields{
			"path":  socketPath,
			"error": err,
		}).Error("Problem with diagnostic socket")
	})
	return err
}

func readDiagnosticInfo(socketPath string) ([]byte, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(conn)
}

// DiagnosticText returns a simple textual output of the agent's status
func (a *Agent) DiagnosticText() string {
	return fmt.Sprintf(
		"SignalFx Agent Status"+
			"\n=====================\n"+
			"\nVersion: %s"+
			"\nAgent Configuration:"+
			"\n%s\n\n"+
			"%s\n"+
			"%s\n"+
			"%s",
		VersionLine,
		utils.IndentLines(config.ToString(a.lastConfig), 2),
		a.writer.DiagnosticText(),
		a.observers.DiagnosticText(),
		a.monitors.DiagnosticText())

}

func (a *Agent) serveInternalMetrics(socketPath string) error {
	if a.internalMetricsSocketStop != nil {
		a.internalMetricsSocketStop()
	}

	if err := os.MkdirAll(filepath.Dir(socketPath), 0700); err != nil {
		return err
	}

	var err error
	a.internalMetricsSocketStop, err = network.RunSimpleSocketServer(socketPath, func(_ net.Conn) string {
		jsonOut, err := json.MarshalIndent(a.InternalMetrics(), "", "  ")
		if err != nil {
			log.WithError(err).Error("Could not serialize internal metrics to JSON")
			return "[]"
		}
		return string(jsonOut)
	}, func(err error) {
		log.WithFields(log.Fields{
			"path":  socketPath,
			"error": err,
		}).Error("Problem with internal metrics socket")
	})
	return err
}

// InternalMetrics aggregates internal metrics from subcomponents and returns a
// list of datapoints that represent the instaneous state of the agent
func (a *Agent) InternalMetrics() []*datapoint.Datapoint {
	out := make([]*datapoint.Datapoint, 0)
	out = append(out, a.writer.InternalMetrics()...)
	out = append(out, a.observers.InternalMetrics()...)
	out = append(out, a.monitors.InternalMetrics()...)

	for i := range out {
		if out[i].Dimensions == nil {
			out[i].Dimensions = make(map[string]string)
		}

		out[i].Dimensions["host"] = a.lastConfig.Hostname
		out[i].Timestamp = time.Now()
	}
	return out
}

func (a *Agent) ensureProfileServerRunning() {
	if !a.profileServerRunning {
		// We don't use that much memory so the default mem sampling rate is
		// too small to be very useful. Setting to 1 profiles ALL allocations
		runtime.MemProfileRate = 1
		// Crank up CPU profile rate too since our CPU usage tends to be pretty
		// bursty around read cycles.
		runtime.SetCPUProfileRate(-1)
		runtime.SetCPUProfileRate(2000)

		go func() {
			a.profileServerRunning = true
			// This is very difficult to access from the host on mac without
			// exposing it on all interfaces
			log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
		}()
	}
}
