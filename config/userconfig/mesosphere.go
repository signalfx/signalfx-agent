package userconfig

import (
	"errors"
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"

	"github.com/spf13/viper"
)

const (
	mesosWorker = "worker"
	mesosMaster = "master"
)

// Mesosphere mesosphere user configuration
type Mesosphere struct {
	Cluster      string
	Role         string
	SystemHealth bool `yaml:"systemHealth,omitempty"`
	Verbose      bool `yaml:"verbose,omitempty"`
}

// LoadYAML loads a yaml file
func (m *Mesosphere) LoadYAML(path string) error {
	var err error
	var file []byte
	// Load the yaml file
	if file, err = ioutil.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(file, &m); err != nil {
			return err
		}
	}
	return err
}

// IsValid determines if the configuration is valid
func (m *Mesosphere) IsValid() (bool, error) {
	if m.Cluster == "" {
		return false, errors.New("mesosphere.cluster must be set")
	}
	if m.Role != mesosMaster && m.Role != mesosWorker {
		return false, errors.New("mesosphere role must be specified")
	}
	return true, nil
}

// Parse parses the configuration into a supplied map
func (m *Mesosphere) Parse(mesos map[string]interface{}) error {
	if ok, err := m.IsValid(); !ok {
		return err
	}

	// Set the cluster name for the mesos default plugin config
	mesos["cluster"] = m.Cluster
	mesos["systemhealth"] = m.SystemHealth
	mesos["verbose"] = m.Verbose
	return nil
}

// ParseDimensions parses the dimensions into the supplied map
func (m *Mesosphere) ParseDimensions(dims map[string]string) error {
	var mesosPort int
	var mesosIDDimName string
	var mesosID string

	client := NewMesosClient(viper.GetViper())
	if m.Role == mesosMaster {
		mesosPort = 5050
		mesosIDDimName = "mesos_master"
	} else if m.Role == mesosWorker {
		mesosIDDimName = "mesos_agent"
		mesosPort = 5051
	}
	if err := client.Configure(viper.GetViper(), mesosPort); err != nil {
		return fmt.Errorf("unable to configure mesos client at configuration time: %s", err)
	}

	ID, _ := client.GetID()
	mesosID = ID.ID

	dims["mesos_cluster"] = m.Cluster
	dims["mesos_role"] = m.Role
	dims[mesosIDDimName] = mesosID
	return nil
}
