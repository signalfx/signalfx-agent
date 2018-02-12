package collectd

//go:generate collectd-template-to-go collectd.conf.tmpl collectd.conf.go

import (
	"bytes"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors/collectd/templating"
	"github.com/signalfx/neo-agent/monitors/types"
	"github.com/signalfx/neo-agent/utils"
)

const (
	// How long to wait for back-to-back (re)starts before actually (re)starting
	restartDelay = 3 * time.Second
)

// Collectd states
const (
	Errored       = "errored"
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
	configMutex sync.Mutex
	conf        *config.CollectdConfig
	// Map of each active monitor to its output instance
	activeMonitors  map[types.MonitorID]types.Output
	genericJMXUsers map[types.MonitorID]bool
	noIDOutputs     map[types.MonitorID]types.Output
	active          bool
	// The port of the active write server, will be 0 if write server isn't
	// started yet.
	writeServerPort int

	// Channels to control the state machine asynchronously
	stop           chan struct{}
	requestRestart chan struct{}
}

var collectdSingleton = &Manager{
	activeMonitors:  make(map[types.MonitorID]types.Output),
	genericJMXUsers: make(map[types.MonitorID]bool),
	noIDOutputs:     make(map[types.MonitorID]types.Output),
	stop:            make(chan struct{}),
	requestRestart:  make(chan struct{}),
}

// Instance returns the singleton instance of the collectd manager
func Instance() *Manager {
	collectdSingleton.configMutex.Lock()
	defer collectdSingleton.configMutex.Unlock()

	if !collectdSingleton.active {
		panic("Don't try and use the collectd manager until it is configured!")
	}

	return collectdSingleton
}

// ConfigureCollectd should be called whenever the main collectd config in the agent
// has changed.  Restarts collectd if the config has changed.
func ConfigureCollectd(conf *config.CollectdConfig) error {
	cm := collectdSingleton

	cm.configMutex.Lock()
	defer cm.configMutex.Unlock()

	if cm.conf == nil || cm.conf.Hash() != conf.Hash() {
		cm.conf = conf

		if !cm.active {
			cm.deleteExistingConfig()

			waitCh := make(chan struct{})
			// This should only have to be called once for the lifetime of the
			// agent.
			go cm.manageCollectd(waitCh)
			// Wait for the write server to be started
			<-waitCh
			cm.active = true
		}

		cm.requestRestart <- struct{}{}
	}

	return nil
}

// ConfigureFromMonitor is how monitors notify the collectd manager that they
// have added a configuration file to managed_config and need a restart. The
// monitorID is passed in so that we can keep track of what monitors are
// actively using collectd.  When a monitor is done (i.e. shutdown) it should
// call MonitorDidShutdown.  GenericJMX monitors should set usesGenericJMX to
// true so that collectd can know to load the java plugin in the collectd.conf
// file so that any JVM config doesn't get set multiple times and cause
// spurious log output.
func (cm *Manager) ConfigureFromMonitor(monitorID types.MonitorID, output types.Output, usesGenericJMX bool, noMonitorID bool) error {
	cm.configMutex.Lock()
	defer cm.configMutex.Unlock()

	cm.activeMonitors[monitorID] = output

	// This is kind of ugly having to keep track of this but it allows us to
	// load the GenericJMX plugin in a central place and then have each
	// GenericJMX monitor render its own config file and not have to worry
	// about reinitializing GenericJMX and causing errors to be thrown.
	if usesGenericJMX {
		cm.genericJMXUsers[monitorID] = true
	}

	// Some legacy collectd plugin configuration might not properly report the
	// monitor ID back to agent, so we have no way of knowing which monitor's
	// Output to associate the datapoints back to.  This is especially true of
	// user provided collectd config templates.  So just stick those outputs
	// into a map so that they can be picked from randomly when we get
	// datapoints without a monitor id.  This is definitely a hack but I can't
	// figure out any generic way to correlate it without requiring users to
	// provide fairly intricate collectd filtering.
	if noMonitorID {
		cm.noIDOutputs[monitorID] = output
	}

	cm.requestRestart <- struct{}{}
	return nil
}

// MonitorDidShutdown should be called by any monitor that uses collectd when
// it is shutdown.
func (cm *Manager) MonitorDidShutdown(monitorID types.MonitorID) {
	cm.configMutex.Lock()
	defer cm.configMutex.Unlock()

	delete(cm.activeMonitors, monitorID)
	delete(cm.genericJMXUsers, monitorID)
	delete(cm.noIDOutputs, monitorID)

	if len(cm.activeMonitors) == 0 && cm.stop != nil {
		close(cm.stop)
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

// WriteServerURL returns the URL of the write server, in case monitors need to
// know it (e.g. the signalfx-metadata plugin).
func (cm *Manager) WriteServerURL() string {
	// Just reuse the config struct's method for making a URL
	conf := *cm.conf
	conf.WriteServerPort = uint16(cm.writeServerPort)
	return conf.WriteServerURL()
}

// ManagedConfigDir returns the directory where monitor config should go.
func (cm *Manager) ManagedConfigDir() string {
	if cm.conf != nil {
		return cm.conf.ManagedConfigDir()
	}
	// This is a programming bug if we get here.
	panic("Collectd must be configured before any monitor tries to use it")
}

// Manage the subprocess with a basic state machine.  This is a bit tricky
// since we have config coming in asynchronously from multiple sources.  This
// function should never return.  waitCh will be closed once the write server
// is setup and right before it is actualy waiting for restart signals.
func (cm *Manager) manageCollectd(waitCh chan<- struct{}) {
	state := Uninitialized
	var cmd *exec.Cmd
	procDied := make(chan struct{})
	restart := make(chan struct{})
	// This is to stop the goroutine that looks for restart requests
	var closeSignal chan struct{}
	var restartDebounced func()
	var restartDebouncedStop chan<- struct{}

	writeServer, err := cm.startWriteServer()
	if err != nil {
		log.WithError(err).Error("Could not start collectd write server")
		state = Errored
	} else {
		cm.writeServerPort = writeServer.RunningPort()
	}

	close(waitCh)

	for {
		log.Debugf("Collectd is now %s", state)

		switch state {

		case Uninitialized:
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
				state = Starting
			}

		case Starting:
			if err := cm.rerenderConf(writeServer.RunningPort()); err != nil {
				log.WithError(err).Error("Could not render collectd.conf")
				state = Errored
				continue
			}

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
			cm.stop = make(chan struct{})

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
	if cm.conf != nil {
		log.Debug("Deleting existing config")
		os.RemoveAll(cm.conf.ConfigDir)
	}
}

func (cm *Manager) startWriteServer() (*WriteHTTPServer, error) {
	writeServer, err := NewWriteHTTPServer(cm.conf.WriteServerIPAddr, cm.conf.WriteServerPort, cm.receiveDPs, cm.receiveEvents)
	if err != nil {
		return nil, err
	}

	if err := writeServer.Start(); err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"ipAddr": cm.conf.WriteServerIPAddr,
		"port":   writeServer.RunningPort(),
	}).Info("Started collectd write server")

	return writeServer, nil
}

func (cm *Manager) receiveDPs(dps []*datapoint.Datapoint) {
	cm.configMutex.Lock()
	defer cm.configMutex.Unlock()

	for i := range dps {
		var monitorID types.MonitorID
		if id, ok := dps[i].Meta["monitorID"].(string); ok {
			monitorID = types.MonitorID(id)
		} else if id := dps[i].Dimensions["monitorID"]; id != "" {
			monitorID = types.MonitorID(id)
			delete(dps[i].Dimensions, "monitorID")
		}

		var output types.Output
		if string(monitorID) == "" {
			// Just get an arbitrary output from the "no id" outputs, doesn't
			// matter which.
			for k := range cm.noIDOutputs {
				output = cm.noIDOutputs[k]
				break
			}
		} else {
			output = cm.activeMonitors[monitorID]
		}

		if output == nil {
			if output == nil {
				log.WithFields(log.Fields{
					"monitorID": monitorID,
				}).Error("Datapoint has an unknown monitorID")
				continue
			}
		}

		output.SendDatapoint(dps[i])
	}
}

func (cm *Manager) receiveEvents(events []*event.Event) {
	cm.configMutex.Lock()
	defer cm.configMutex.Unlock()

	for i := range events {
		var monitorID types.MonitorID
		if id, ok := events[i].Properties["monitorID"].(string); ok {
			monitorID = types.MonitorID(id)
			delete(events[i].Properties, "monitorID")
		} else if id := events[i].Dimensions["monitorID"]; id != "" {
			monitorID = types.MonitorID(id)
			delete(events[i].Dimensions, "monitorID")
		}

		if string(monitorID) == "" {
			log.WithFields(log.Fields{
				"event": spew.Sdump(events[i]),
			}).Error("Event does not have a monitorID as either a dimension or property field, cannot send")
			continue
		}

		output := cm.activeMonitors[monitorID]
		if output == nil {
			log.WithFields(log.Fields{
				"monitorID": monitorID,
			}).Error("Event has an unknown monitorID, cannot send")
			continue
		}

		output.SendEvent(events[i])
	}
}

func (cm *Manager) rerenderConf(writeHTTPPort int) error {
	output := bytes.Buffer{}

	log.WithFields(log.Fields{
		"context": cm.conf,
	}).Debug("Rendering main collectd.conf template")

	// Copy so that hash of config struct is consistent
	conf := *cm.conf
	conf.HasGenericJMXMonitor = len(cm.genericJMXUsers) > 0
	conf.WriteServerPort = uint16(writeHTTPPort)

	if err := CollectdTemplate.Execute(&output, &conf); err != nil {
		return errors.Wrapf(err, "Failed to render collectd template")
	}

	return templating.WriteConfFile(output.String(), cm.conf.ConfigFilePath())
}

func (cm *Manager) makeChildCommand() *exec.Cmd {
	cmd := exec.Command("lib64/ld-linux-x86-64.so.2", "bin/collectd", "-f", "-C", cm.conf.ConfigFilePath())

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// This is Linux-specific and will cause collectd to be killed by the OS if
	// the agent dies
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}

	return cmd
}
