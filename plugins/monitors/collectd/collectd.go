package collectd

// #cgo CFLAGS: -I/usr/include/collectd -I/usr/include -I/usr/local/include/collectd -I/usr/local/include -DSIGNALFX_EIM=1
// #cgo LDFLAGS: /usr/local/lib/collectd/libcollectd.so
// #include <stdint.h>
// #include <stdlib.h>
// #include "collectd.h"
import "C"
import (
	"errors"
	"fmt"
	"log"
	"time"

	"path/filepath"

	"io/ioutil"

	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/plugins/monitors/collectd/config"
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
	state        string
	services     services.ServiceInstances
	templatesDir string
	confFile     string
	pluginsDir   string
}

// NewCollectd constructor
func NewCollectd(name string, config *viper.Viper) (*Collectd, error) {
	plugin, err := plugins.NewPlugin(name, config)
	if err != nil {
		return nil, err
	}

	// Convert to absolute paths since our cwd can get changed.
	templatesDir := plugin.Config.GetString("templatesdir")
	if templatesDir == "" {
		return nil, errors.New("configuration missing templatesDir entry")
	}
	templatesDir, err = filepath.Abs(templatesDir)
	if err != nil {
		return nil, err
	}

	confFile := plugin.Config.GetString("conffile")
	if confFile == "" {
		return nil, errors.New("configuration missing confFile entry")
	}

	confFile, err = filepath.Abs(confFile)
	if err != nil {
		return nil, err
	}

	pluginsDir := plugin.Config.GetString("pluginsDir")
	if err != nil {
		return nil, err
	}

	return &Collectd{plugin, Stopped, nil, templatesDir, confFile, pluginsDir}, nil
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
		servicePlugins, err := collectd.createPluginsFromServices(services)
		if err != nil {
			return err
		}

		plugins, err := collectd.getStaticPlugins()
		if err != nil {
			return err
		}

		collectd.services = services

		plugins = append(plugins, servicePlugins...)
		if err := collectd.writePlugins(plugins); err != nil {
			return err
		}

		collectd.reload()
	}

	return nil
}

// reload reloads collectd configuration
func (collectd *Collectd) reload() {
	log.Println("reloading collectd plugins")
	collectd.state = Reloading

	C.reload()

	for {
		if int(C.is_reloading()) == 1 {
			break
		} else {
			time.Sleep(time.Duration(1) * time.Second)
		}
	}

	collectd.state = Running
}

// writePlugins takes a list of plugin instances and generates a collectd.conf
// formatted configuration.
func (collectd *Collectd) writePlugins(plugins []*config.Plugin) error {
	config, err := config.RenderCollectdConf(collectd.pluginsDir, collectd.templatesDir, &config.AppConfig{
		AgentConfig: config.NewCollectdConfig(),
		Plugins:     plugins,
	})

	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(collectd.confFile, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write collectd config: %s", err)
	}

	return nil
}

// getStaticPlugins returns a list of plugins specified in the agent config
func (collectd *Collectd) getStaticPlugins() ([]*config.Plugin, error) {
	static := struct {
		StaticPlugins []struct {
			Name   string
			Type   string
			Config map[string]interface{}
		}
	}{}

	if err := collectd.Config.Unmarshal(&static); err != nil {
		return nil, err
	}

	var plugins []*config.Plugin

	for _, plugin := range static.StaticPlugins {
		pluginInstance, err := config.NewPlugin(services.ServiceType(plugin.Type), plugin.Name)
		if err != nil {
			return nil, err
		}

		if err := config.LoadPluginConfig(map[string]interface{}{"config": plugin.Config},
			plugin.Type, pluginInstance); err != nil {
			return nil, err
		}

		plugins = append(plugins, pluginInstance)
	}

	return plugins, nil
}

func (collectd *Collectd) createPluginsFromServices(sis services.ServiceInstances) ([]*config.Plugin, error) {
	log.Printf("Configuring collectd plugins for %+v", sis)
	var plugins []*config.Plugin

	for _, service := range sis {
		log.Printf("reconfiguring collectd service: %s (%s)", service.Service.Name, service.Service.Type)

		// TODO: Name is not unique, not sure what to use here.
		plugin, err := config.NewPlugin(service.Service.Type, service.Service.Name)
		if err != nil {
			log.Printf("unsupported service %s for collectd", service.Service.Type)
			continue
		}

		plugin.Host = service.Port.IP
		plugin.Port = service.Port.PrivatePort

		plugins = append(plugins, plugin)
	}

	fmt.Printf("Configured plugins: %+v\n", plugins)

	return plugins, nil
}

// Start collectd monitoring
func (collectd *Collectd) Start() (err error) {
	println("starting collectd")
	if collectd.state == Running {
		return errors.New("already running")
	}

	collectd.services = nil

	log.Println("configuring static collectd plugins before first start")
	plugins, err := collectd.getStaticPlugins()
	if err != nil {
		return err
	}

	if err := collectd.writePlugins(plugins); err != nil {
		return err
	}

	go func() {
		cConfFile := C.CString(collectd.confFile)
		defer C.free(cConfFile)
		C.start(nil, cConfFile)
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
