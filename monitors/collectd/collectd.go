package collectd

//go:generate collectd-template-to-go collectd.conf.tmpl collectd.conf.go

import (
	"bytes"
	"os"
	"os/exec"
	"reflect"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/fatih/set.v0"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/config/types"
	"github.com/signalfx/neo-agent/monitors/collectd/templating"
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
	state    string
	confFile string
	// triggers a reload of the collectd daemon
	reloadChan           chan int
	stopChan             chan int
	configMutex          sync.Mutex
	stateMutex           sync.Mutex
	cmdMutex             sync.Mutex
	cmd                  *exec.Cmd
	conf                 *config.CollectdConfig
	restartDebounced     func()
	restartDebouncedStop chan<- struct{}
	activeMonitors       map[types.MonitorID]bool
}

var collectdSingleton = &Manager{
	state:          Stopped,
	reloadChan:     make(chan int),
	stopChan:       make(chan int),
	activeMonitors: make(map[types.MonitorID]bool),
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
				cm.reloadChan <- 1
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
func (cm *Manager) ConfigureFromMonitor(monitorID types.MonitorID, conf *config.CollectdConfig) bool {
	cm.configMutex.Lock()
	defer cm.configMutex.Unlock()

	cm.activeMonitors[monitorID] = true

	// Delete existing config on the first call
	if cm.conf == nil {
		cm.deleteExistingConfig()
	}

	if reflect.DeepEqual(conf, cm.conf) {
		return true
	}

	if !cm.validateConfig(conf) {
		return false
	}

	cm.conf = conf
	cm.rerenderConf()

	cm.Restart()
	return true
}

func (cm *Manager) validateConfig(conf *config.CollectdConfig) bool {
	valid := true

	if !validLogLevels.Has(conf.LogLevel) {
		log.WithFields(log.Fields{
			"validLevels": validLogLevels.String(),
			"level":       conf.LogLevel,
		}).Error("Invalid collectd log level")
		valid = false
	}

	return valid
}

// Shutdown collectd and all associated resources
func (cm *Manager) Shutdown() {
	log.Debug("Shutting down collectd")
	if cm.State() != Stopped {
		cm.stopChan <- 0
		cm.restartDebouncedStop <- struct{}{}
		cm.restartDebouncedStop = nil
	}
}

// MonitorDidShutdown should be called by any monitor that uses collectd when
// it is shutdown.
func (cm *Manager) MonitorDidShutdown(monitorID types.MonitorID) {
	cm.configMutex.Lock()
	defer cm.configMutex.Unlock()

	delete(cm.activeMonitors, monitorID)
	if len(cm.activeMonitors) == 0 {
		cm.Shutdown()
	} else {
		cm.Restart()
	}
}

// State for collectd monitoring
func (cm *Manager) State() string {
	cm.stateMutex.Lock()
	defer cm.stateMutex.Unlock()

	log.Infof("Setting collectd state to %s", cm.state)
	return cm.state
}

// setState sets state for collectd monitoring
func (cm *Manager) setState(state string) {
	cm.stateMutex.Lock()
	defer cm.stateMutex.Unlock()

	cm.state = state
}

func (cm *Manager) rerenderConf() bool {
	output := bytes.Buffer{}

	log.WithFields(log.Fields{
		"context": cm.conf,
	}).Debug("Rendering main collectd.conf template")

	if err := CollectdTemplate.Execute(&output, cm.conf); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to render collectd template")
		return false
	}

	return templating.WriteConfFile(output.String(), collectdConfPath)
}

func (cm *Manager) runCollectd() {
	stoppedCh := make(chan struct{}, 1)
	restartDelay := 2 * time.Second

	stop := func() {
		cm.setState(ShuttingDown)
		cm.killChildProc()
		<-stoppedCh
		cm.setState(Stopped)
	}

	go cm.runAsChildProc(stoppedCh)

	for {
		select {
		case <-cm.stopChan:
			log.Info("Stopping collectd")
			stop()
			return
		case <-cm.reloadChan:
			stop()
			log.Info("Collectd stopped, restarting")
			go cm.runAsChildProc(stoppedCh)
		case <-stoppedCh:
			if cm.state == Running {
				log.Error("Collectd died when it was supposed to be running, restarting...")
			} else {
				log.Warn("Collectd stopped in an unexpected way")
				continue
			}
			time.Sleep(restartDelay)
			go cm.runAsChildProc(stoppedCh)
		}
	}
}

func (cm *Manager) killChildProc() {
	cm.cmdMutex.Lock()
	if cm.cmd.Process != nil {
		log.Info("Killing old Collectd process")
		cm.cmd.Process.Kill()
		cm.cmd.Wait()
	}
	cm.cmdMutex.Unlock()
}

func (cm *Manager) runAsChildProc(stoppedCh chan<- struct{}) {
	log.Info("Starting Collectd child process")

	cm.cmdMutex.Lock()
	cm.cmd = exec.Command("collectd", "-f", "-C", collectdConfPath)

	cm.cmd.Stdout = os.Stdout
	cm.cmd.Stderr = os.Stderr

	err := cm.cmd.Start()
	if err != nil {
		log.WithError(err).Error("Could not start Collectd child process!")
		stoppedCh <- struct{}{}
		return
	}

	cm.setState(Running)

	cm.cmdMutex.Unlock()
	cm.cmd.Wait()

	stoppedCh <- struct{}{}
}

// Delete existing config in case there were plugins configured before that won't
// be configured on this run.
func (cm *Manager) deleteExistingConfig() {
	log.Debug("Deleting existing config")
	os.RemoveAll(managedConfigDir)
	os.Remove(collectdConfPath)
}
