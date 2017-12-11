// Package neopy holds the logic for managing Python plugins using a subprocess
// running Python. Currently there is only a DataDog monitor type, but we will
// support collectd Python plugins.  Communiation between this program and the
// Python runner is done through ZeroMQ IPC sockets.
//
// These Python monitors are configured the same as other monitors.
package neopy

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/neo-agent/monitors"
	log "github.com/sirupsen/logrus"
)

// NeoPy is the adapter to the NeoPy Python monitor runner process.  It
// communiates with Python using ZeroMQ IPC sockets.  Each general type of
// Python plugin (e.g. Datadog, collectd, etc.) should get its own generic
// monitor type that uses this adapter.
type NeoPy struct {
	subproc        *exec.Cmd
	subprocCancel  func()
	registered     bool
	shouldShutdown bool
	dpChannels     map[monitors.MonitorID]chan<- *datapoint.Datapoint
	dpChannelsLock sync.RWMutex
	mainDPChan     <-chan *DatapointMessage

	// Used to request monitor registrations
	registerQueue *RegisterQueue
	// Used to send configurations to python plugins
	configQueue *ConfigQueue
	// Used to tell individual monitors to shutdown when no longer needed
	shutdownQueue *ShutdownQueue
	// Used by NeoPy to send metrics back to NeoAgent
	datapointQueue *DatapointsQueue
	// Used by NeoPy to send log entries back to NeoAgent
	loggingQueue *LoggingQueue
}

var neoPySingleton = newInstance()

// Instance returns the singleton NeoPy adapter instance
func Instance() *NeoPy {
	return neoPySingleton
}

func newInstance() *NeoPy {
	return &NeoPy{
		shouldShutdown: false,
		dpChannels:     make(map[monitors.MonitorID]chan<- *datapoint.Datapoint),
		registerQueue:  newRegisterQueue(),
		configQueue:    newConfigQueue(),
		shutdownQueue:  newShutdownQueue(),
		datapointQueue: newDatapointsQueue(),
		loggingQueue:   newLoggingQueue(),
	}
}

func (npy *NeoPy) start() error {
	log.Info("Starting NeoPy")

	npy.mainDPChan = npy.datapointQueue.listenForDatapoints()

	if err := npy.registerQueue.start(); err != nil {
		npy.Shutdown()
		return err
	}

	if err := npy.configQueue.start(); err != nil {
		npy.Shutdown()
		return err
	}

	if err := npy.shutdownQueue.start(); err != nil {
		npy.Shutdown()
		return err
	}

	if err := npy.datapointQueue.start(); err != nil {
		npy.Shutdown()
		return err
	}

	if err := npy.loggingQueue.start(); err != nil {
		npy.Shutdown()
		return err
	}

	go npy.sendDatapointsToMonitors()

	//ctx, cancel := context.WithCancel(context.Background())
	//npy.subprocCancel = cancel
	//go npy.runAsChildProc(ctx)
	npy.subproc = &exec.Cmd{}

	return nil
}

// Configure could be used later for passing config to the NeoPy instance.
// Right now it is just used to start up the child process if not running.
func (npy *NeoPy) Configure() bool {
	if npy.subproc == nil {
		npy.start()
	}
	return true
}

// EnsureMonitorsRegistered will ask the Python subproc for a list of
// monitors that it supports.  It then registers those monitors using the
// standard register callback that all monitors use.
func (npy *NeoPy) EnsureMonitorsRegistered() {
	if npy.registered {
		return
	}

	monitorTypes := npy.registerQueue.getMonitorTypeList()
	for _, _type := range monitorTypes {
		log.WithFields(log.Fields{
			"monitorType": _type,
		}).Debug("Registering Python monitor")
		if strings.HasPrefix(_type, DDMonitorTypePrefix) {
			registerDDCheck(_type)
		}
	}
	npy.registered = true
}

func (npy *NeoPy) sendDatapointsForMonitorTo(monitorID monitors.MonitorID, dpChan chan<- *datapoint.Datapoint) {
	npy.dpChannelsLock.Lock()
	defer npy.dpChannelsLock.Unlock()
	npy.dpChannels[monitorID] = dpChan
}

func (npy *NeoPy) sendDatapointsToMonitors() {
	for {
		select {
		case dpMessage := <-npy.mainDPChan:
			npy.dpChannelsLock.RLock()

			if ch, ok := npy.dpChannels[dpMessage.MonitorID]; ok {
				ch <- dpMessage.Datapoint
			} else {
				log.WithFields(log.Fields{
					"dpMessage": dpMessage,
				}).Error("Could not find monitor ID to send datapoint from NeoPy")
			}

			npy.dpChannelsLock.RUnlock()
		}
	}
}

// ConfigureInPython sends the given config to the python subproc and returns
// whether configuration was successful
func (npy *NeoPy) ConfigureInPython(config interface{}) bool {
	return npy.configQueue.configure(config)
}

// ShutdownMonitor will shutdown the given monitor id in the python subprocess
func (npy *NeoPy) ShutdownMonitor(monitorID monitors.MonitorID) {
	npy.shutdownQueue.sendShutdownForMonitor(monitorID)
	delete(npy.dpChannels, monitorID)
}

// Shutdown the whole NeoPy child process, not just individual monitors
func (npy *NeoPy) Shutdown() {
	npy.shouldShutdown = true
	if npy.subprocCancel != nil {
		npy.subprocCancel()
	}
}

func (npy *NeoPy) runAsChildProc(ctx context.Context) {
	restartDelay := 2 * time.Second
	for {
		log.Info("Starting NeoPy child process")
		cmd := exec.CommandContext(ctx,
			"/usr/bin/env", "PYTHONPATH=/usr/local/lib", "python", "-m", "neopy",
			"--register-path", npy.registerQueue.socketPath(),
			"--configure-path", npy.configQueue.socketPath(),
			"--shutdown-path", npy.shutdownQueue.socketPath(),
			"--datapoints-path", npy.datapointQueue.socketPath(),
			"--logging-path", npy.loggingQueue.socketPath())

		err := cmd.Run()
		// This should always be set whenever we call the cancel func
		// corresponding to the `ctx` so that we can know whether the proc died
		// on purpose or not.
		if npy.shouldShutdown {
			return
		}

		log.WithFields(log.Fields{
			"error": err,
		}).Error("NeoPy child process died, restarting")

		time.Sleep(restartDelay)
	}
}
