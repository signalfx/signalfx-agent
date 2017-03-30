package collectd

// #cgo CFLAGS: -I/usr/include/collectd -I/usr/include -I/usr/local/include/collectd -I/usr/local/include -DSIGNALFX_EIM=1
// #cgo LDFLAGS: /usr/local/lib/collectd/libcollectd.so
// #include <stdint.h>
// #include <stdlib.h>
// #include <string.h>
// #include "collectd.h"
// #include "configfile.h"
// #include "plugin.h"
import "C"
import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"path/filepath"

	"io/ioutil"

	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/plugins/monitors/collectd/config"
	"github.com/signalfx/neo-agent/services"
	"github.com/spf13/viper"
)

const (
	pluginType = "monitors/collectd"

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
	state         string
	services      services.Instances
	templatesDirs []string
	confFile      string
	pluginsDir    string
	reloadChan    chan int
	stopChan      chan int
	configMutex   sync.Mutex
	stateMutex    sync.Mutex
	configDirty   bool
}

func init() {
	plugins.Register(pluginType, NewCollectd)
}

// NewCollectd constructor
func NewCollectd(name string, config *viper.Viper) (plugins.IPlugin, error) {
	plugin, err := plugins.NewPlugin(name, pluginType, config)
	if err != nil {
		return nil, err
	}
	c := &Collectd{
		Plugin:     plugin,
		state:      Stopped,
		reloadChan: make(chan int),
		stopChan:   make(chan int)}
	if err := c.load(plugin.Config); err != nil {
		return nil, err
	}

	return c, nil
}

// load collectd plugin config from the provided config
func (collectd *Collectd) load(config *viper.Viper) error {
	var err error

	templatesDirs := config.GetStringSlice("templatesdirs")
	if len(templatesDirs) == 0 {
		return errors.New("config missing templatesDirs entry")
	}

	// Convert to absolute paths since our cwd can get changed.
	for i := range templatesDirs {
		absTemplateDir, err := filepath.Abs(templatesDirs[i])
		if err != nil {
			return err
		}
		templatesDirs[i] = absTemplateDir
	}

	confFile := config.GetString("conffile")
	if confFile == "" {
		return errors.New("config missing confFile entry")
	}

	confFile, err = filepath.Abs(confFile)
	if err != nil {
		return err
	}

	pluginsDir := config.GetString("pluginsDir")
	if err != nil {
		return err
	}

	// Set values once everything passes muster.
	collectd.templatesDirs = templatesDirs
	collectd.confFile = confFile
	collectd.pluginsDir = pluginsDir

	return nil
}

// Monitor services from collectd monitor
func (collectd *Collectd) Write(services services.Instances) error {
	collectd.configMutex.Lock()
	defer collectd.configMutex.Unlock()

	changed := false

	if collectd.configDirty {
		changed = true
		collectd.configDirty = false
	}

	if changed {
		log.Print("reloading config due to dirty flag")
	} else if len(collectd.services) != len(services) {
		changed = true
	} else {
		for i := range services {
			// Checks if the services are either completely different (i.e. a
			// service has been added or removed) or if the service's
			// configuration has changed such as mapping to a different
			// template.
			if !services[i].Equivalent(&collectd.services[i]) {
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
	if collectd.State() == Running {
		collectd.setState(Reloading)
		collectd.reloadChan <- 1
		for collectd.State() == Reloading {
			time.Sleep(time.Duration(1) * time.Second)
		}
	}
}

// writePlugins takes a list of plugin instances and generates a collectd.conf
// formatted configuration.
func (collectd *Collectd) writePlugins(plugins []*config.Plugin) error {
	collectdConfig := config.NewCollectdConfig()
	// If this is empty then collectd determines hostname.
	collectdConfig.Hostname = viper.GetString("hostname")

	// group instances by plugin
	instancesMap := config.GroupByPlugin(plugins)

	config, err := config.RenderCollectdConf(collectd.pluginsDir, collectd.templatesDirs, &config.AppConfig{
		AgentConfig: collectdConfig,
		Plugins:     instancesMap,
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
		// This could possibly be cleaned up to use inline annotation but
		// haven't figured out how to make it work.
		StaticPlugins map[string]map[string]interface{}
	}{}

	if err := collectd.Config.Unmarshal(&static); err != nil {
		return nil, err
	}

	var plugins []*config.Plugin

	for pluginName, plugin := range static.StaticPlugins {
		yamlPluginType, ok := plugin["plugin"]
		if !ok {
			return nil, fmt.Errorf("static plugin %s missing plugin type", pluginName)
		}

		pluginType, ok := yamlPluginType.(string)
		if !ok {
			return nil, fmt.Errorf("static plugin %s is not a string", pluginName)
		}

		pluginInstance, err := config.NewPlugin(services.ServiceType(pluginType), pluginName)
		if err != nil {
			return nil, err
		}

		if err := config.LoadPluginConfig(map[string]interface{}{"config": plugin},
			pluginType, pluginInstance); err != nil {
			return nil, err
		}

		plugins = append(plugins, pluginInstance)
	}

	return plugins, nil
}

func (collectd *Collectd) createPluginsFromServices(sis services.Instances) ([]*config.Plugin, error) {
	log.Printf("Configuring collectd plugins for %+v", sis)
	var plugins []*config.Plugin

	for _, service := range sis {
		// TODO: Name is not unique, not sure what to use here.
		plugin, err := config.NewPlugin(service.Service.Type, service.Service.Name)
		if err != nil {
			log.Printf("unsupported service %s for collectd", service.Service.Type)
			continue
		}

		for key, val := range service.Orchestration.Dims {
			dim := key + "=" + val
			if len(plugin.Dims) > 0 {
				plugin.Dims = plugin.Dims + ","
			}
			plugin.Dims = plugin.Dims + dim
		}

		plugin.Host = service.Port.IP
		if service.Orchestration.PortPref == services.PRIVATE {
			plugin.Port = service.Port.PrivatePort
		} else {
			plugin.Port = service.Port.PublicPort
		}

		if plugin.Port == 0 {
			continue
		}

		log.Printf("reconfiguring collectd service: %s (%s) to use template %s", service.Service.Name, service.Service.Type, service.Config)

		plugin.Template = service.Config

		plugins = append(plugins, plugin)
	}

	fmt.Printf("Configured plugins: %+v\n", plugins)

	return plugins, nil
}

// Start collectd monitoring
func (collectd *Collectd) Start() (err error) {
	log.Println("starting collectd")

	if collectd.State() == Running {
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

	go collectd.run()

	return nil
}

// Stop collectd monitoring
func (collectd *Collectd) Stop() {
	if collectd.State() != Stopped {
		collectd.stopChan <- 0
	}
}

// Reload collectd config
func (collectd *Collectd) Reload(config *viper.Viper) error {
	collectd.configMutex.Lock()
	defer collectd.configMutex.Unlock()

	if err := collectd.load(config); err != nil {
		return err
	}

	collectd.Config = config
	collectd.configDirty = true
	return nil
}

// GetWatchDirs returns list of directories that when changed will trigger reload
func (collectd *Collectd) GetWatchDirs(config *viper.Viper) []string {
	return config.GetStringSlice("templatesdirs")
}

// State for collectd monitoring
func (collectd *Collectd) State() string {
	collectd.stateMutex.Lock()
	state := collectd.state
	collectd.stateMutex.Unlock()
	return state
}

// setState sets state for collectd monitoring
func (collectd *Collectd) setState(state string) {
	collectd.stateMutex.Lock()
	collectd.state = state
	collectd.stateMutex.Unlock()
}

func (collectd *Collectd) run() {
	collectd.setState(Running)
	defer collectd.setState(Stopped)

	// TODO - global collectd interval should be configurable
	interval := time.Duration(10 * time.Second)
	cConfFile := C.CString(collectd.confFile)
	defer C.free(cConfFile)

	C.plugin_init_ctx()

	C.cf_read(cConfFile)

	C.init_collectd()
	C.interval_g = C.cf_get_default_interval()

	C.plugin_init_all()

	for {
		t := time.Now()

		C.plugin_read_all()

		remainingTime := interval - time.Since(t)
		if remainingTime > 0 {
			time.Sleep(remainingTime)
		}

		select {
		case <-collectd.stopChan:
			log.Println("stop collectd requested")
			C.plugin_shutdown_all()
			return
		case <-collectd.reloadChan:
			log.Println("reload collectd plugins requested")
			collectd.setState(Reloading)

			C.plugin_shutdown_for_reload()
			C.plugin_init_ctx()
			C.cf_read(cConfFile)
			C.plugin_init_for_reload()

			collectd.setState(Running)
		}
	}
}
