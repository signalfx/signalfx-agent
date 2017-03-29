package file

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/services"
	"github.com/spf13/viper"
)

const (
	pluginType = "observers/file"
)

// File observer plugin
type File struct {
	plugins.Plugin
	path string
}

func init() {
	plugins.Register(pluginType, NewFile)
}

// NewFile constructor
func NewFile(name string, config *viper.Viper) (plugins.IPlugin, error) {
	plugin, err := plugins.NewPlugin(name, pluginType, config)
	if err != nil {
		return nil, err
	}
	config.SetDefault("path", "/etc/signalfx/service_instances.json");
	return &File{plugin, config.GetString("path")}, nil
}

// Discover services from a file
func (file *File) Read() (services.Instances, error) {

	if _, err := os.Stat(file.path); err == nil {

		var instances *services.Instances

		jsonContent, err := ioutil.ReadFile(file.path)
		if err != nil {
			return nil, fmt.Errorf("reading %s failed: %s", file.path, err)
		}

		if err := json.Unmarshal(jsonContent, &instances); err != nil {
			return nil, fmt.Errorf("unmarshaling %s failed: %s", file.path, err)
		}

		sort.Sort(*instances)
		return *instances, nil
	}

	return make(services.Instances, 0), nil
}
