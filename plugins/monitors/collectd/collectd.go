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
	"os"
	"sync"
	"time"
	"unsafe"

	"path/filepath"

	cfg "github.com/signalfx/neo-agent/config"
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
	state       string
	services    services.Instances
	assetSyncer *cfg.AssetSyncer
	assets      *cfg.AssetsView
	confFile    string
	pluginsDir  string
	reloadChan  chan int
	stopChan    chan int
	configMutex sync.Mutex
	stateMutex  sync.Mutex
	configDirty bool
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
	return &Collectd{
		Plugin:      plugin,
		state:       Stopped,
		reloadChan:  make(chan int),
		stopChan:    make(chan int),
		assetSyncer: cfg.NewAssetSyncer()}, nil
}

// load collectd plugin config from the provided config
func (collectd *Collectd) load(config *viper.Viper) error {
	var err error

	builtins := config.GetString("templates.builtins")
	if builtins == "" {
		return errors.New("config missing templates.builtins entry")
	}

	overrides := config.GetString("templates.overrides")

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
	ws := cfg.NewAssetSpec()
	ws.Dirs["builtins"] = builtins
	if overrides != "" {
		ws.Dirs["overrides"] = overrides
	}
	if err := collectd.assetSyncer.Update(ws); err != nil {
		return err
	}

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
	if collectd.assets == nil {
		return errors.New("collectd plugin assets not yet loaded")
	}
	builtins, ok := collectd.assets.Dirs["builtins"]
	if !ok {
		return fmt.Errorf("builtins missing for collectd")
	}

	overrides := collectd.assets.Dirs["overrides"]

	collectdConfig := config.NewCollectdConfig()
	// If this is empty then collectd determines hostname.
	collectdConfig.Hostname = viper.GetString("hostname")

	// Set other collectd configurations from viper if they're set
	if viper.IsSet("plugins.collectd.interval") {
		collectdConfig.Interval = uint(viper.GetInt("plugins.collectd.interval"))
	}
	if viper.IsSet("plugins.collectd.readThreads") {
		collectdConfig.ReadThreads = uint(viper.GetInt("plugins.collectd.readThreads"))
	}
	if viper.IsSet("plugins.collectd.writeQueueLimitHigh") {
		collectdConfig.WriteQueueLimitHigh = uint(viper.GetInt("plugins.collectd.writeQueueLimitHigh"))
	}
	if viper.IsSet("plugins.collectd.writeQueueLimitLow") {
		collectdConfig.WriteQueueLimitLow = uint(viper.GetInt("plugins.collectd.writeQueueLimitLow"))
	}
	if viper.IsSet("plugins.collectd.timeout") {
		collectdConfig.Timeout = uint(viper.GetInt("plugins.collectd.timeout"))
	}
	if viper.IsSet("plugins.collectd.collectInternalStats") {
		collectdConfig.CollectInternalStats = viper.GetBool("plugins.collectd.collectInternalStats")
	}

	// group instances by plugin
	instancesMap := config.GroupByPlugin(plugins)

	config, err := config.RenderCollectdConf(collectd.pluginsDir, builtins, overrides,
		&config.AppConfig{
			AgentConfig: collectdConfig,
			Plugins:     instancesMap,
		})
	if err != nil {
		return err
	}

	f, err := os.Create(collectd.confFile)
	if err != nil {
		return fmt.Errorf("failed to truncate collectd config: %s", err)
	}
	defer f.Close()

	_, err = f.Write([]byte(config))
	if err != nil {
		return fmt.Errorf("failed to write collectd config: %s", err)
	}

	// We need to sync here since collectd might be restarted very quickly
	// after writing this.
	err = f.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync collectd config file to disk: %s", err)
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

		pluginInstance, err := config.NewPlugin(config.PluginType(pluginType), pluginName)
		if err != nil {
			return nil, err
		}

		// This block takes configurations for a static plugin and makes the
		// available in templates under ".Vars"
		// TODO: Look for sensetivity to camel casing
		// in yaml files (staticPlugins vs staticplugins)
		if err := config.LoadPluginConfig(map[string]interface{}{"vars": plugin},
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
		plugin, err := config.NewInstancePlugin(service.Service.Plugin, service.Service.Name)
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

		plugin.Template = service.Template
		for k, v := range service.Vars {
			plugin.Vars[k] = v
		}

		plugins = append(plugins, plugin)
	}

	fmt.Printf("Configured plugins: %+v\n", plugins)

	return plugins, nil
}

// Start collectd monitoring
func (collectd *Collectd) Start() (err error) {
	log.Println("starting collectd")

	collectd.assetSyncer.Start(func(view *cfg.AssetsView) {
		collectd.configMutex.Lock()
		defer collectd.configMutex.Unlock()

		log.Printf("assets changed for collectd, setting configDirty to true")

		collectd.assets = view
		collectd.configDirty = true
	})

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
		// Assets may not be loaded yet, will be rewritten when they're ready.
		log.Printf("unable to write plugins on start: %s", err)
	}

	go collectd.run()

	return nil
}

// Stop collectd monitoring
func (collectd *Collectd) Stop() {
	collectd.assetSyncer.Stop()

	if collectd.State() != Stopped {
		collectd.stopChan <- 0
	}
}

// Configure collectd config
func (collectd *Collectd) Configure(config *viper.Viper) error {
	collectd.configMutex.Lock()
	defer collectd.configMutex.Unlock()

	if err := collectd.load(config); err != nil {
		return err
	}

	collectd.Config = config
	collectd.configDirty = true
	return nil
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
	// See https://blog.golang.org/c-go-cgo#TOC_2. 
	defer C.free(unsafe.Pointer(cConfFile))

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
