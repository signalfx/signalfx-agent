// This observer is primarily meant for development and test purposes.  It will
// watch a json file, which should consist of an array of serialized service
// instances.
package file

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/observers"
	log "github.com/sirupsen/logrus"
)

const (
	observerType = "file"
)

var logger = log.WithFields(log.Fields{"observerType": observerType})

type Config struct {
	config.ObserverConfig
	Path string `default:"/etc/signalfx/service_instances.json"`
}

// File observer plugin
type File struct {
	serviceCallbacks *observers.ServiceCallbacks
	serviceDiffer    *observers.ServiceDiffer
	config           *Config
}

func init() {
	observers.Register(observerType, func(cbs *observers.ServiceCallbacks) interface{} {
		return &File{
			serviceCallbacks: cbs,
		}
	}, &Config{})
}

// Configure the docker client
func (file *File) Configure(config *Config) bool {
	file.config = config
	file.serviceDiffer = &observers.ServiceDiffer{
		DiscoveryFn:     file.discover,
		IntervalSeconds: 5,
		Callbacks:       file.serviceCallbacks,
	}
	file.serviceDiffer.Start()

	return file.config == nil
}

// Discover services from a file
func (file *File) discover() []*observers.ServiceInstance {
	if _, err := os.Stat(file.config.Path); err != nil {
		return nil
	}

	var instances []*observers.ServiceInstance

	jsonContent, err := ioutil.ReadFile(file.config.Path)
	if err != nil {
		logger.WithFields(log.Fields{
			"error":    err,
			"filePath": file.config.Path,
		}).Error("Could not read service file")
		return nil
	}

	if err := json.Unmarshal(jsonContent, &instances); err != nil {
		logger.WithFields(log.Fields{
			"error":    err,
			"filePath": file.config.Path,
		}).Error("Could not parse service json")
	}

	return instances
}

func (file *File) Shutdown() {
	file.serviceDiffer.Stop()
}
