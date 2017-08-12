package collectd

//go:generate collectd-template-to-go collectd.conf.tmpl collectd.conf.go

// #cgo CFLAGS: -I/usr/src/collectd/src/daemon -I/usr/src/collectd/src -I/usr/local/include -DSIGNALFX_EIM=1
// #cgo LDFLAGS: /usr/local/lib/libcollectd.so
// #include <stdint.h>
// #include <stdlib.h>
// #include <string.h>
// #include "collectd.h"
// #include "configfile.h"
// #include "plugin.h"
import "C"
import (
	"bytes"
	"os"
	"reflect"
	"sync"
	"time"
	"unsafe"

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
	// Reloading collectd plugins
	Reloading = "reloading"
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
	cm.setState(Running)
	defer cm.setState(Stopped)

	cConfFile := C.CString(collectdConfPath)
	// See https://blog.golang.org/c-go-cgo#TOC_2.
	defer C.free(unsafe.Pointer(cConfFile))

	C.plugin_init_ctx()

	C.cf_read(cConfFile)

	C.init_collectd()
	C.interval_g = C.cf_get_default_interval()

	C.plugin_init_all()

	for {
		C.plugin_read_all()

		select {
		case <-cm.stopChan:
			log.Info("Stopping Collectd")
			C.plugin_shutdown_all()
			cm.setState(Stopped)
			return
		case <-cm.reloadChan:
			log.Info("Restarting Collectd")
			cm.setState(Reloading)

			C.plugin_shutdown_for_reload()
			C.plugin_init_ctx()
			C.cf_read(cConfFile)
			C.plugin_init_for_reload()

			cm.setState(Running)
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
