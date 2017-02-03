package collectd

// #cgo CFLAGS: -I/usr/include/collectd -I/usr/include -I/usr/local/include/collectd -I/usr/local/include -DSIGNALFX_EIM=1
// #cgo LDFLAGS: /usr/local/lib/collectd/libcollectd.so
// #include <stdint.h>
// #include <stdlib.h>
// #include "collectd.h"
import "C"
import (
	"errors"
	"log"
	"time"

	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/services"
	"github.com/spf13/viper"
)

const (
	// Running collectd
	Running = "running"
	// Stopped collectd
	Stopped = "stopped"
	// Reloading collectd plugins
	Reloading = "reloading"
)

// Collectd Monitor
type Collectd struct {
	plugins.Plugin
	state    string
	services services.ServiceInstances
}

// NewCollectd constructor
func NewCollectd(name string, config *viper.Viper) (*Collectd, error) {
	plugin, err := plugins.NewPlugin(name, config)
	if err != nil {
		return nil, err
	}
	return &Collectd{plugin, Stopped, nil}, nil
}

// Monitor services from collectd monitor
func (collectd *Collectd) Write(services services.ServiceInstances) error {
	changed := false
	if len(collectd.services) != len(services) {
		changed = true
	} else {
		for i := range services {
			if services[i].ID != collectd.services[i].ID {
				changed = true
				break
			}
		}
	}

	if changed {
		if err := collectd.configurePlugins(services); err != nil {
			return err
		}
		collectd.state = Reloading

		C.reload()

		for {
			if int(C.is_reloading()) == 1 {
				break
			} else {
				time.Sleep(time.Duration(1) * time.Second)
			}
		}
		collectd.services = services
		collectd.state = Running
	}

	return nil
}

func (collectd *Collectd) configurePlugins(services services.ServiceInstances) error {
	// TODO - print services to configure for now
	// use service.Name as key to configuration mapping
	for _, service := range services {
		log.Printf("reconfiguring collectd service: %s", service.Service.Name)
	}
	return nil
}

// Start collectd monitoring
func (collectd *Collectd) Start() (err error) {
	println("starting collectd")
	if collectd.state == Running {
		return errors.New("already running")
	}

	collectd.services = make(services.ServiceInstances, 0)

	go func() {
		confFile := C.CString("collectd.conf")
		defer C.free(confFile)
		C.start(nil, confFile)
	}()

	log.Print("Collectd started")
	collectd.state = Running
	return nil
}

// Stop collectd monitoring
func (collectd *Collectd) Stop() {
	C.stop()
	collectd.state = Stopped
	collectd.services = nil
}

// Status for collectd monitoring
func (collectd *Collectd) Status() string {
	return collectd.state
}
