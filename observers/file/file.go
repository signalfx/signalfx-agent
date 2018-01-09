// Package file is a file-based observer that is primarily meant for
// development and test purposes.  It will watch a json file, which should
// consist of an array of serialized service instances.
package file

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/observers"
	log "github.com/sirupsen/logrus"
)

const (
	observerType = "file"
)

var logger = log.WithFields(log.Fields{"observerType": observerType})

// Config for the file observer
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
func (file *File) Configure(config *Config) error {
	file.config = config

	if file.serviceDiffer != nil {
		file.serviceDiffer.Stop()
	}

	file.serviceDiffer = &observers.ServiceDiffer{
		DiscoveryFn:     file.discover,
		IntervalSeconds: 5,
		Callbacks:       file.serviceCallbacks,
	}
	file.serviceDiffer.Start()

	return nil
}

// Discover services from a file
func (file *File) discover() []services.Endpoint {
	if _, err := os.Stat(file.config.Path); err != nil {
		return nil
	}

	var instances []*services.ContainerEndpoint

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

	var out []services.Endpoint
	for i := range instances {
		out = append(out, services.Endpoint(instances[i]))
	}
	return out
}

// Shutdown the service differ routine
func (file *File) Shutdown() {
	if file.serviceDiffer != nil {
		file.serviceDiffer.Stop()
	}
}
