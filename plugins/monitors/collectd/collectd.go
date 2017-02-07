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
	"github.com/signalfx/neo-agent/plugins/monitors"
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
	state       string
	services    services.ServiceInstances
	servicesDRS []services.ServiceDiscoveryRuleset
}

// NewCollectd constructor
func NewCollectd(config *viper.Viper) (*Collectd, error) {
	plugin, err := plugins.NewPlugin(monitors.Collectd, config)
	if err != nil {
		return nil, err
	}
	return &Collectd{plugin, Stopped, nil, nil}, nil
}

// Monitor services from collectd monitor
func (collectd *Collectd) Monitor(services services.ServiceInstances) error {

	// let this monitor determine which services are applicable here
	applicableServices, err := collectd.getApplicableServices(services)
	if err != nil {
		return err
	}

	changed := false
	if len(collectd.services) != len(applicableServices) {
		changed = true
	} else {
		for i := range applicableServices {
			if applicableServices[i].ID != collectd.services[i].ID {
				changed = true
				break
			}
		}
	}

	if changed {
		if err := collectd.configurePlugins(applicableServices); err != nil {
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
		collectd.services = applicableServices
		collectd.state = Running
	}

	return nil
}

func (collectd *Collectd) getApplicableServices(sis services.ServiceInstances) (services.ServiceInstances, error) {
	applicableServices := make(services.ServiceInstances, 0, len(sis))
	if collectd.servicesDRS != nil {
		for i := range sis {
			for _, ruleset := range collectd.servicesDRS {
				matches, err := sis[i].Matches(ruleset)
				if err != nil {
					return nil, err
				}

				if matches {
					// set service name to ruleset name and add as service to monitor
					sis[i].Service.Name = ruleset.Name
					sis[i].Service.Type = ruleset.Type
					applicableServices = append(applicableServices, sis[i])
					break
				}
			}
		}
	}
	return applicableServices, nil
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
	if collectd.state == Running {
		return errors.New("already running")
	}

	collectd.services = make(services.ServiceInstances, 0)

	if servicesFile := collectd.Config.GetString("servicesfile"); servicesFile != "" {
		log.Printf("loading service discovery signatures from %s", servicesFile)
		lsignatures, err := services.LoadServiceSignatures(servicesFile)
		if err != nil {
			return err
		}
		collectd.servicesDRS = lsignatures.Signatures
	}

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
func (collectd *Collectd) Stop() error {
	C.stop()
	collectd.state = Stopped
	collectd.services = nil
	collectd.servicesDRS = nil
	return nil
}

// Status for collectd monitoring
func (collectd *Collectd) Status() string {
	return collectd.state
}
