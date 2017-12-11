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
	"gopkg.in/fatih/set.v0"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd/templating"
	"github.com/signalfx/neo-agent/monitors/collectd/write"
	"github.com/signalfx/neo-agent/utils"
)

const (
	pluginType       = "monitors/collectd"
	collectdConfPath = "/etc/collectd/collectd.conf"
	managedConfigDir = "/etc/collectd/managed_config/"

	// How long to wait for back-to-back (re)starts before actually (re)starting
	restartDebounceDuration = 3 * time.Second

	// Running collectd
	Running = "running"
	// Stopped collectd
	Stopped = "stopped"
	// ShuttingDown collectd
	ShuttingDown = "shutting-down"
	// Restarting collectd
	Restarting = "restarting"
)

var validLogLevels = set.NewNonTS("debug", "info", "notice", "warning", "err")

// Manager coordinates the collectd conf file and running the embedded collectd
// library.
type Manager struct {
	state                string
	confFile             string
	stoppedCh            chan struct{}
	configMutex          sync.Mutex
	stateMutex           sync.Mutex
	cmdMutex             sync.Mutex
	cmd                  *exec.Cmd
	conf                 *config.CollectdConfig
	restartDebounced     func()
	restartDebouncedStop chan<- struct{}
	activeMonitors       map[monitors.MonitorID]bool
	genericJMXUsers      map[monitors.MonitorID]bool
	// The local server that collectd sends its datapoints to
	writeServer *write.Server
}

var collectdSingleton = &Manager{
	state:           Stopped,
	activeMonitors:  make(map[monitors.MonitorID]bool),
	genericJMXUsers: make(map[monitors.MonitorID]bool),
}

// Instance returns the singleton instance of the collectd manager
func Instance() *Manager {
	return collectdSingleton
}

// Restart collectd, or start it if it hasn't been.  The restart will be
// "debounced" so that it will not happen immediately upon the first request,
// but will wait for `restartDebounceDuration` in case multiple monitors
// request a restart.  Unfortunately we don't have any way of selectively
// restarting certain plugins at this point.
func (cm *Manager) Restart() {
	if cm.restartDebounced == nil {
		cm.restartDebounced, cm.restartDebouncedStop = utils.Debounce0(func() {
			if cm.State() == Stopped {
				log.Info("Starting collectd")
				go cm.runCollectd()
			} else {
				cm.reload()
			}
		}, restartDebounceDuration)
	}

	log.Debug("Queueing Collectd (re)start")
	cm.restartDebounced()
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

	cm.activeMonitors[monitorID] = true

	// This is kind of ugly having to keep track of this but it allows us to
	// load the GenericJMX plugin in a central place and then have each
	// GenericJMX monitor render its own config file and not have to worry
	// about reinitializing GenericJMX and causing errors to be thrown.
	if usesGenericJMX {
		cm.genericJMXUsers[monitorID] = true
	}

	// Delete existing config on the first call
	if cm.conf == nil {
		cm.deleteExistingConfig()
	}

	if err := cm.validateConfig(conf); err != nil {
		return err
	}

	cm.conf = conf
	cm.rerenderConf()

	err := cm.ensureWriteServerRunning(conf.WriteServerIPAddr, conf.WriteServerPort, dpChan, eventChan)
	if err != nil {
		return errors.Wrap(err, "Could not start up collectd write server")
	}

	cm.Restart()
	return nil
}

func (cm *Manager) ensureWriteServerRunning(ipAddr string, port uint16, dpChan chan<- *datapoint.Datapoint, eventChan chan<- *event.Event) error {
	if cm.writeServer == nil {
		var err error
		cm.writeServer, err = write.NewServer(ipAddr, port, dpChan, eventChan)
		if err != nil {
			return err
		}
		log.WithFields(log.Fields{"ipAddr": ipAddr, "port": port}).Info("Started collectd write server")
	}

	return nil
}

func (cm *Manager) validateConfig(conf *config.CollectdConfig) error {
	if !validLogLevels.Has(conf.LogLevel) {
		return errors.Errorf("Invalid collectd log level %s.  Valid choices are %v",
			conf.LogLevel,
			validLogLevels)
	}

	return nil
}

// State for collectd monitoring
func (cm *Manager) State() string {
	cm.stateMutex.Lock()
	defer cm.stateMutex.Unlock()

	return cm.state
}

// setState sets state for collectd monitoring
func (cm *Manager) setState(state string) {
	cm.stateMutex.Lock()
	defer cm.stateMutex.Unlock()

	cm.state = state
	log.Infof("Setting collectd state to %s", cm.state)
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

func (cm *Manager) runCollectd() {
	cm.stoppedCh = make(chan struct{}, 1)
	restartDelay := 2 * time.Second

	for {
		cm.runAsChildProc()

		if cm.state != Running {
			break
		}

		log.Error("Collectd died when it was supposed to be running, restarting...")
		time.Sleep(restartDelay)
	}

	close(cm.stoppedCh)
}

func (cm *Manager) runAsChildProc() {
	log.Info("Starting Collectd child process")

	cm.cmdMutex.Lock()
	cm.cmd = exec.Command("collectd", "-f", "-C", collectdConfPath)

	cm.cmd.Stdout = os.Stdout
	cm.cmd.Stderr = os.Stderr

	err := cm.cmd.Start()
	if err != nil {
		log.WithError(err).Error("Could not start collectd child process!")
		return
	}

	cm.setState(Running)

	cm.cmdMutex.Unlock()
	cm.cmd.Wait()
}

func (cm *Manager) stop() {
	if cm.state != Running {
		log.Error("Collectd was told to stop but isn't running")
		return
	}

	cm.setState(ShuttingDown)
	cm.killChildProc()
	<-cm.stoppedCh

	cm.setState(Stopped)
}

func (cm *Manager) reload() {
	log.Info("Reloading collectd")
	cm.stop()
	log.Info("Collectd stopped, restarting")
	go cm.runCollectd()
}

func (cm *Manager) killChildProc() {
	cm.cmdMutex.Lock()
	defer cm.cmdMutex.Unlock()

	if cm.cmd.Process != nil {
		cm.cmd.Process.Kill()
		cm.cmd.Wait()
		log.Info("Old collectd process killed")
	}
}

// Delete existing config in case there were plugins configured before that won't
// be configured on this run.
func (cm *Manager) deleteExistingConfig() {
	log.Debug("Deleting existing config")
	os.RemoveAll(managedConfigDir)
	os.Remove(collectdConfPath)
}

// MonitorDidShutdown should be called by any monitor that uses collectd when
// it is shutdown.
func (cm *Manager) MonitorDidShutdown(monitorID monitors.MonitorID) {
	cm.configMutex.Lock()
	defer cm.configMutex.Unlock()

	delete(cm.activeMonitors, monitorID)
	delete(cm.genericJMXUsers, monitorID)
	if len(cm.activeMonitors) == 0 {
		cm.Shutdown()
	} else {
		cm.rerenderConf()
		cm.Restart()
	}
}

// Shutdown collectd and all associated resources
func (cm *Manager) Shutdown() {
	log.Debug("Shutting down collectd")
	if cm.State() != Stopped {
		cm.stop()
		cm.restartDebouncedStop <- struct{}{}
		cm.restartDebouncedStop = nil
		cm.restartDebounced = nil
	}

	cm.conf = nil

	if cm.writeServer != nil {
		if err := cm.writeServer.Close(); err != nil {
			log.WithError(err).Warn("Could not shutdown collectd write server")
		} else {
			log.Info("Shut down collectd write server")
		}
		cm.writeServer = nil
	}
}
