package collectd

//go:generate collectd-template-to-go collectd.conf.tmpl collectd.conf.go

import (
	"bytes"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd/templating"
	"github.com/signalfx/neo-agent/monitors/collectd/write"
	"github.com/signalfx/neo-agent/utils"
)

const (
	pluginType = "monitors/collectd"

	collectdConfPath = "./tmp/collectd.conf"
	managedConfigDir = "./tmp/managed_config/"

	// How long to wait for back-to-back (re)starts before actually (re)starting
	restartDelay = 3 * time.Second
)

// Collectd states
const (
	Errored       = "errored"
	Initializing  = "initializing"
	Restarting    = "restarting"
	Running       = "running"
	Starting      = "starting"
	Stopped       = "stopped"
	ShuttingDown  = "shutting-down"
	Uninitialized = "uninitialized"
)

// Manager coordinates the collectd conf file and running the embedded collectd
// library.
type Manager struct {
	configMutex     sync.Mutex
	conf            *config.CollectdConfig
	activeMonitors  map[monitors.MonitorID]bool
	genericJMXUsers map[monitors.MonitorID]bool
	dpChan          chan<- *datapoint.Datapoint
	eventChan       chan<- *event.Event
	active          bool

	// Channels to control the state machine asynchronously
	stop           chan struct{}
	requestRestart chan struct{}
}

var collectdSingleton = &Manager{
	activeMonitors:  make(map[monitors.MonitorID]bool),
	genericJMXUsers: make(map[monitors.MonitorID]bool),
	stop:            make(chan struct{}),
	requestRestart:  make(chan struct{}),
}

// Instance returns the singleton instance of the collectd manager
func Instance() *Manager {
	if !collectdSingleton.active {
		// This should only have to be called once for the lifetime of the
		// agent.
		go collectdSingleton.manageCollectd()
		collectdSingleton.active = true
	}
	return collectdSingleton
}

// ConfigureFromMonitor configures collectd, renders the collectd.conf file,
// and queues a (re)start.  Individual collectd-based monitors write their own
// config files and should queue restarts when they have rendered their own
// config files.  The monitorID is passed in so that we can keep track of what
// monitors are actively using collectd.  When a monitor is done (i.e.
// shutdown) it should call MonitorDidShutdown.
func (cm *Manager) ConfigureFromMonitor(monitorID monitors.MonitorID, conf *config.CollectdConfig,
	dpChan chan<- *datapoint.Datapoint, eventChan chan<- *event.Event, usesGenericJMX bool) error {

	cm.configMutex.Lock()
	defer cm.configMutex.Unlock()

	cm.conf = conf
	cm.dpChan = dpChan
	cm.eventChan = eventChan

	cm.activeMonitors[monitorID] = true

	// This is kind of ugly having to keep track of this but it allows us to
	// load the GenericJMX plugin in a central place and then have each
	// GenericJMX monitor render its own config file and not have to worry
	// about reinitializing GenericJMX and causing errors to be thrown.
	if usesGenericJMX {
		cm.genericJMXUsers[monitorID] = true
	}

	cm.requestRestart <- struct{}{}
	return nil
}

// MonitorDidShutdown should be called by any monitor that uses collectd when
// it is shutdown.
func (cm *Manager) MonitorDidShutdown(monitorID monitors.MonitorID) {
	cm.configMutex.Lock()
	defer cm.configMutex.Unlock()

	delete(cm.activeMonitors, monitorID)
	delete(cm.genericJMXUsers, monitorID)

	if len(cm.activeMonitors) == 0 {
		cm.stop <- struct{}{}
	} else {
		cm.requestRestart <- struct{}{}
	}
}

// RequestRestart should be used to indicate that a configuration in
// managed_config has been updated (e.g. by a monitor) and that collectd needs
// to restart.  This method will not immediately restart but will wait for a
// bit to batch together multiple back-to-back restarts.
func (cm *Manager) RequestRestart() {
	cm.requestRestart <- struct{}{}
}

// Manage the subprocess with a basic state machine.  This is a bit tricky
// since we have config coming in asynchronously from multiple sources.  This
// function should never return.
func (cm *Manager) manageCollectd() {
	state := Uninitialized
	var cmd *exec.Cmd
	procDied := make(chan struct{})
	restart := make(chan struct{})
	// This is to stop the goroutine that looks for restart requests
	var closeSignal chan struct{}
	var restartDebounced func()
	var restartDebouncedStop chan<- struct{}
	var writeServer *write.Server

	for {
		log.Debugf("Collectd is now %s", state)

		switch state {

		case Uninitialized:
			collectdSingleton.deleteExistingConfig()

			closeSignal = make(chan struct{})
			restartDebounced, restartDebouncedStop = utils.Debounce0(func() {
				restart <- struct{}{}
			}, restartDelay)

			go func() {
				for {
					select {
					case <-cm.requestRestart:
						restartDebounced()
					case <-closeSignal:
						return
					}
				}
			}()

			// Block here until we actually get a start request
			select {
			case <-restart:
				state = Initializing
			}

		case Initializing:
			var err error
			writeServer, err = cm.startWriteServer()
			if err != nil {
				log.WithError(err).Error("Could not start collectd write server")
				state = Errored
				continue
			}

			state = Starting

		case Starting:
			cm.rerenderConf()

			cmd = cm.makeChildCommand()

			if err := cmd.Start(); err != nil {
				log.WithError(err).Error("Could not start collectd child process!")
				time.Sleep(restartDelay)
				state = Starting
				continue
			}

			go func() {
				cmd.Wait()
				procDied <- struct{}{}
			}()

			go func() {
				select {
				case <-cm.requestRestart:
					restartDebounced()
				}
			}()

			state = Running

		case Running:
			select {
			case <-restart:
				state = Restarting
			case <-cm.stop:
				state = ShuttingDown
			case <-procDied:
				log.Error("Collectd died when it was supposed to be running, restarting...")
				time.Sleep(restartDelay)
				state = Starting
			}

		case Restarting:
			cmd.Process.Kill()
			<-procDied
			state = Starting

		case ShuttingDown:
			cmd.Process.Kill()
			<-procDied
			state = Stopped

		case Stopped:
			writeServer.Close()
			restartDebouncedStop <- struct{}{}
			close(closeSignal)
			state = Uninitialized

		// If you go to the Errored state, make sure nothing is left running!
		case Errored:
			log.Error("Collectd is in an error state, waiting for config change")
			state = Uninitialized
		}

	}
}

// Delete existing config in case there were plugins configured before that won't
// be configured on this run.
func (cm *Manager) deleteExistingConfig() {
	log.Debug("Deleting existing config")
	os.RemoveAll(managedConfigDir)
	os.Remove(collectdConfPath)
}

func (cm *Manager) startWriteServer() (*write.Server, error) {
	writeServer, err := write.NewServer(cm.conf.WriteServerIPAddr, cm.conf.WriteServerPort, cm.dpChan, cm.eventChan)
	if err != nil {
		return nil, err
	}
	log.WithFields(log.Fields{
		"ipAddr": cm.conf.WriteServerIPAddr,
		"port":   cm.conf.WriteServerPort,
	}).Info("Started collectd write server")

	return writeServer, nil
}

func (cm *Manager) rerenderConf() error {
	output := bytes.Buffer{}

	log.WithFields(log.Fields{
		"context": cm.conf,
	}).Debug("Rendering main collectd.conf template")

	cm.conf.HasGenericJMXMonitor = len(cm.genericJMXUsers) > 0

	if err := CollectdTemplate.Execute(&output, cm.conf); err != nil {
		return errors.Wrapf(err, "Failed to render collectd template")
	}

	return templating.WriteConfFile(output.String(), collectdConfPath)
}

func (cm *Manager) makeChildCommand() *exec.Cmd {
	cmd := exec.Command("lib64/ld-linux-x86-64.so.2", "bin/collectd", "-f", "-C", collectdConfPath)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd
}
