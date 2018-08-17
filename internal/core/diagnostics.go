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
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/network/simpleserver"
	log "github.com/sirupsen/logrus"
)

// VersionLine should be populated by the startup logic to contain version
// information that can be reported in diagnostics.
var VersionLine string

// Serves the diagnostic status on the specified path
func (a *Agent) serveDiagnosticInfo(path string) error {
	if a.diagnosticServerStop != nil {
		a.diagnosticServerStop()
	}

	if runtime.GOOS != "windows" {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
	}

	var err error
	a.diagnosticServerStop, err = simpleserver.Run(path, func(_ net.Conn) string {
		return a.DiagnosticText()
	}, func(err error) {
		log.WithFields(log.Fields{
			"path":  path,
			"error": err,
		}).Error("Problem with diagnostic socket")
	})
	return err
}

func readDiagnosticInfo(path string) ([]byte, error) {
	conn, err := simpleserver.Dial(path)
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

func (a *Agent) serveInternalMetrics(path string) error {
	if a.internalMetricsServerStop != nil {
		a.internalMetricsServerStop()
	}

	if runtime.GOOS != "windows" {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
	}

	var err error
	a.internalMetricsServerStop, err = simpleserver.Run(path, func(_ net.Conn) string {
		jsonOut, err := json.MarshalIndent(a.InternalMetrics(), "", "  ")
		if err != nil {
			log.WithError(err).Error("Could not serialize internal metrics to JSON")
			return "[]"
		}
		return string(jsonOut)
	}, func(err error) {
		log.WithFields(log.Fields{
			"path":  path,
			"error": err,
		}).Error("Problem with internal metrics socket")
	})
	return err
}

// InternalMetrics aggregates internal metrics from subcomponents and returns a
// list of datapoints that represent the instaneous state of the agent
func (a *Agent) InternalMetrics() []*datapoint.Datapoint {
	out := make([]*datapoint.Datapoint, 0)

	mstat := runtime.MemStats{}
	runtime.ReadMemStats(&mstat)
	out = append(out, []*datapoint.Datapoint{
		sfxclient.Cumulative("sfxagent.go_total_alloc", nil, int64(mstat.TotalAlloc)),
		sfxclient.Gauge("sfxagent.go_sys", nil, int64(mstat.Sys)),
		sfxclient.Cumulative("sfxagent.go_mallocs", nil, int64(mstat.Mallocs)),
		sfxclient.Cumulative("sfxagent.go_frees", nil, int64(mstat.Frees)),
		sfxclient.Gauge("sfxagent.go_heap_alloc", nil, int64(mstat.HeapAlloc)),
		sfxclient.Gauge("sfxagent.go_heap_sys", nil, int64(mstat.HeapSys)),
		sfxclient.Gauge("sfxagent.go_heap_idle", nil, int64(mstat.HeapIdle)),
		sfxclient.Gauge("sfxagent.go_heap_inuse", nil, int64(mstat.HeapInuse)),
		sfxclient.Gauge("sfxagent.go_heap_released", nil, int64(mstat.HeapReleased)),
		sfxclient.Gauge("sfxagent.go_stack_inuse", nil, int64(mstat.StackInuse)),
		sfxclient.Gauge("sfxagent.go_next_gc", nil, int64(mstat.NextGC)),
		sfxclient.Cumulative("sfxagent.go_pause_total_ns", nil, int64(mstat.PauseTotalNs)),
		sfxclient.Cumulative("sfxagent.go_gc_cpu_fraction", nil, int64(mstat.GCCPUFraction)),
		sfxclient.Gauge("sfxagent.go_num_gc", nil, int64(mstat.NumGC)),
		sfxclient.Gauge("sfxagent.go_gomaxprocs", nil, int64(runtime.GOMAXPROCS(0))),
		sfxclient.Gauge("sfxagent.go_num_goroutine", nil, int64(runtime.NumGoroutine())),
	}...)

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
