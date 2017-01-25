package collectd

// #cgo CFLAGS: -I/usr/include/collectd -I/usr/include -DSIGNALFX_EIM=1
// #cgo LDFLAGS: /usr/lib/collectd/libcollectd.so
// #include <stdint.h>
// #include "collectd.h"
import "C"
import (
	"errors"
	"log"
	"time"

	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/services"
)

// Collectd Monitor
type Collectd struct {
	plugins.Plugin
	state    string
	services services.ServiceInstances
}

// NewCollectd constructor
func NewCollectd(configuration map[string]string) *Collectd {
	return &Collectd{plugins.NewPlugin("collectd", configuration), "stopped", make(services.ServiceInstances, 0)}
}

// Monitor services from collectd monitor
func (collectd *Collectd) Monitor(services services.ServiceInstances) error {

	// temporary basic change detection (reconfigure/reload plugins on any change)
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
		if err := collectd.configurePlugins(services); err == null {
			return err
		}

		C.reload()

		for {
			if int(C.is_reloading()) == 1 {
				break
			} else {
				log.Print("waiting for reload to complete")
				time.Sleep(time.Duration(1) * time.Second)
			}
		}
		collectd.services = services
	}

	return nil
}

func (collectd *Collectd) configurePlugins(services services.ServiceInstances) error {
	log.Print("reconfiguring collectd plugins")
	return nil
}

// Start collectd monitoring
func (collectd *Collectd) Start() error {
	if collectd.state == "running" {
		return errors.New("already running")
	}

	go C.start()

	log.Print("Collectd started")
	collectd.state = "running"
	return nil
}

// Stop collectd monitoring
func (collectd *Collectd) Stop() error {
	C.stop()
	collectd.state = "stopped"
	return nil
}

// Status for collectd monitoring
func (collectd *Collectd) Status() string {
	return collectd.state
}
