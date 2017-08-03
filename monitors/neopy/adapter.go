package neopy

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/neo-agent/core/config"
	log "github.com/sirupsen/logrus"
)

type NeoPy struct {
	subproc        *exec.Cmd
	subprocCancel  func()
	registered     bool
	shouldShutdown bool
	dpChannels     map[config.MonitorID]chan<- *datapoint.Datapoint
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
}

var neoPySingleton = NewInstance()

func GetInstance() *NeoPy {
	return neoPySingleton
}

func NewInstance() *NeoPy {

	return &NeoPy{
		shouldShutdown: false,
		dpChannels:     make(map[config.MonitorID]chan<- *datapoint.Datapoint),
		registerQueue:  NewRegisterQueue(),
		// Used to send configurations to python plugins
		configQueue: NewConfigQueue(),
		// Used to tell individual monitors to shutdown when no longer needed
		shutdownQueue: NewShutdownQueue(),
		// Used by NeoPy to send metrics back to NeoAgent
		datapointQueue: NewDatapointsQueue(),
	}
}

func (npy *NeoPy) start() error {
	log.Info("Starting NeoPy")

	npy.mainDPChan = npy.datapointQueue.ListenForDatapoints()

	if err := npy.registerQueue.Start(); err != nil {
		npy.Shutdown()
		return err
	}

	if err := npy.configQueue.Start(); err != nil {
		npy.Shutdown()
		return err
	}

	if err := npy.shutdownQueue.Start(); err != nil {
		npy.Shutdown()
		return err
	}

	if err := npy.datapointQueue.Start(); err != nil {
		npy.Shutdown()
		return err
	}

	go npy.sendDatapointsToMonitors()

	//ctx, cancel := context.WithCancel(context.Background())
	//npy.subprocCancel = cancel
	//go npy.runAsChildProc(ctx)

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

func (npy *NeoPy) EnsureMonitorsRegistered() {
	if npy.registered {
		return
	}

	monitorTypes := npy.registerQueue.GetMonitorList()
	for _, _type := range monitorTypes {
		log.WithFields(log.Fields{
			"monitorType": _type,
		}).Debug("Registering Python monitor")
		if strings.HasPrefix(_type, DDMonitorTypePrefix) {
			RegisterDDCheck(_type)
		}
	}
	npy.registered = true
}

func (npy *NeoPy) SendDatapointsForMonitorTo(monitorId config.MonitorID, dpChan chan<- *datapoint.Datapoint) {
	npy.dpChannelsLock.Lock()
	defer npy.dpChannelsLock.Unlock()
	npy.dpChannels[monitorId] = dpChan
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

func (npy *NeoPy) ConfigureInPython(config interface{}) bool {
	return npy.configQueue.Configure(config)
}

func (npy *NeoPy) ShutdownMonitor(monitorId config.MonitorID) {
	delete(npy.dpChannels, monitorId)
	npy.shutdownQueue.SendShutdownForMonitor(monitorId)
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
			"/usr/bin/env", "python", "-m", "neopy",
			"--register-path", npy.registerQueue.SocketPath(),
			"--configure-path", npy.configQueue.SocketPath(),
			"--shutdown-path", npy.shutdownQueue.SocketPath(),
			"--datapoints-path", npy.datapointQueue.SocketPath())

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
