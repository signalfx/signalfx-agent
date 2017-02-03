package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/signalfx/neo-agent/pipelines"
	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/plugins/filters"
	"github.com/signalfx/neo-agent/plugins/filters/services"
	"github.com/signalfx/neo-agent/plugins/monitors"
	"github.com/signalfx/neo-agent/plugins/monitors/collectd"
	"github.com/signalfx/neo-agent/plugins/observers"
	"github.com/signalfx/neo-agent/plugins/observers/docker"
	"github.com/signalfx/neo-agent/utils"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
)

var (
	// Version for agent
	Version string
	// BuiltTime for the agent
	BuiltTime string
	// CollectdVersion embedded in agent
	CollectdVersion string
)

// DefaultInterval is used if not configured
const DefaultInterval = 10

// Agent for monitoring host/service metrics
type Agent struct {
	// Interval to observer service activity
	Interval int
	plugins  []plugins.IPlugin
	pipeline *pipelines.Pipeline
}

// NewAgent with defaults
func NewAgent() *Agent {
	return &Agent{DefaultInterval, nil, nil}
}

// pluginConfig is just a holder for a plugin name and type when loading
type pluginConfig struct {
	Name string
	Type string
}

// Loads subconfigs from configuration file. The given name should be a map
// whose keys are the name of a plugin. That map should itself have a key `type`
// that is the plugin type.
func loadSubConfigs(name string) (map[pluginConfig]*viper.Viper, error) {
	sub := viper.Sub(name)
	if sub == nil {
		return nil, fmt.Errorf("no %ss have been configured", name)
	}

	var keys []string

	for _, key := range sub.AllKeys() {
		idx := strings.Index(key, ".")
		if idx < 1 {
			return nil, fmt.Errorf("key %s is missing '.'", key)
		}
		keys = append(keys, key[0:idx])
	}

	keys = utils.UniqueStrings(keys)
	ret := map[pluginConfig]*viper.Viper{}

	for _, key := range keys {
		viperKey := fmt.Sprintf("%s.%s", name, key)
		s := viper.Sub(viperKey)
		if s == nil {
			return nil, fmt.Errorf("missing key %s", viperKey)
		}

		typ := s.GetString("type")
		if typ == "" {
			return nil, fmt.Errorf("%s is missing type", viperKey)
		}

		config := s.Sub("configuration")
		ret[pluginConfig{key, typ}] = config
	}

	return ret, nil
}

// Configure an agent using configuration file
func (agent *Agent) Configure(configfile string) error {
	viper.SetDefault("interval", DefaultInterval)

	viper.SetConfigFile(configfile)
	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	agent.Interval = viper.GetInt("interval")

	loaded, err := loadSubConfigs("observers")
	if err != nil {
		return err
	}

	for plugin, configuration := range loaded {
		switch plugin.Type {
		case observers.Docker:
			if observer, err := docker.NewDocker(plugin.Name, configuration); err == nil {
				agent.plugins = append(agent.plugins, observer)
			} else {
				return err
			}
		}
	}

	loaded, err = loadSubConfigs("monitors")
	if err != nil {
		return err
	}

	for plugin, configuration := range loaded {
		switch plugin.Type {
		case monitors.Collectd:
			if monitor, err := collectd.NewCollectd(plugin.Name, configuration); err == nil {
				agent.plugins = append(agent.plugins, monitor)
			} else {
				return err
			}
		}
	}

	loaded, err = loadSubConfigs("filters")
	if err != nil {
		return err
	}

	for plugin, configuration := range loaded {
		switch plugin.Type {
		case filters.ServiceRules:
			if filter, err := services.NewRuleFilter(plugin.Name, configuration); err == nil {
				agent.plugins = append(agent.plugins, filter)
			} else {
				return err
			}
		}
	}

	if agent.pipeline, err = pipelines.NewPipeline("default", viper.GetStringSlice("pipeline.default"), agent.plugins); err != nil {
		return err
	}

	return nil
}

func main() {
	var agentConfig = flag.String("config", "/etc/signalfx/agent.yaml", "agent config file")
	var version = flag.Bool("version", false, "print agent version")

	flag.Parse()

	if *version {
		fmt.Printf("agent-version: %s, collectd-version: %s, built-time: %s\n", Version, CollectdVersion, BuiltTime)
		os.Exit(0)
	}

	cwc, cancel := context.WithCancel(context.Background())

	agent := NewAgent()
	if err := agent.Configure(*agentConfig); err != nil {
		log.Printf("failed to configure agent: %s", err)
		os.Exit(1)
	}

	exitCh := make(chan struct{})
	go func(ctx context.Context) {
		log.Print("agent started")

		for _, plugin := range agent.plugins {
			log.Printf("starting plugin %s", plugin.String())
			if err := plugin.Start(); err != nil {
				log.Printf("failed to start plugin %s: %s", plugin.String(), err)
			}
		}

		for {
			log.Print("executing pipeline")

			if err := agent.pipeline.Execute(); err != nil {
				log.Printf("pipeline execute failed: %s", err)
			}

			select {
			case <-ctx.Done():
				for _, plugin := range agent.plugins {
					log.Printf("stopping plugin %s", plugin.String())
					plugin.Stop()
				}
				exitCh <- struct{}{}
				return
			default:
			}

			time.Sleep(time.Duration(agent.Interval) * time.Second)
		}
	}(cwc)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	go func() {
		select {
		case <-signalCh:
			log.Print("stopping agent ...")
			cancel()
			return
		}
	}()
	<-exitCh
}
