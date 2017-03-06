package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/signalfx/neo-agent/pipelines"
	"github.com/signalfx/neo-agent/plugins"
	_ "github.com/signalfx/neo-agent/plugins/all"
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

	// envReplacer replaces . and - with _
	envReplacer = strings.NewReplacer(".", "_", "-", "_")
)

const (
	// DefaultInterval is used if not configured
	DefaultInterval = 10
	// DefaultPipeline is used if not configured
	DefaultPipeline = "docker"

	// envPrefix is the environment variable prefix
	envPrefix      = "SFX"
	envMergeConfig = "SFX_MERGE_CONFIG"
)

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

// Configure an agent using configuration file
func (agent *Agent) Configure(configfile string) error {
	viper.SetDefault("interval", DefaultInterval)
	viper.SetDefault("pipeline", DefaultPipeline)

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(envReplacer)
	viper.SetEnvPrefix(envPrefix)
	viper.SetConfigFile(configfile)

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	if merge := os.Getenv(envMergeConfig); len(merge) > 1 {
		for _, mergeFile := range strings.Split(merge, ":") {
			if file, err := os.Open(mergeFile); err == nil {
				defer file.Close()
				log.Printf("merging config %s", mergeFile)
				if err := viper.MergeConfig(file); err != nil {
					return err
				}
			}
		}
	}

	agent.Interval = viper.GetInt("interval")

	// Load plugins.
	pluginsConfig := viper.GetStringMap("plugins")

	for pluginName := range pluginsConfig {
		pluginType := viper.GetString(fmt.Sprintf("plugins.%s.plugin", pluginName))

		if len(pluginType) < 1 {
			return fmt.Errorf("plugin %s missing plugin key", pluginName)
		}

		if create, ok := plugins.Plugins[pluginType]; ok {
			log.Printf("loading plugin %s (%s)", pluginType, pluginName)

			config := viper.Sub("plugins." + pluginName)
			// This allows a configuration variable foo.bar to be overridable by
			// SFX_FOO_BAR=value.
			config.AutomaticEnv()
			config.SetEnvKeyReplacer(envReplacer)
			config.SetEnvPrefix(strings.ToUpper(
				fmt.Sprintf("%s_plugins_%s", envPrefix, pluginName)))

			pluginInst, err := create(pluginName, config)
			if err != nil {
				return err
			}

			agent.plugins = append(agent.plugins, pluginInst)
		} else {
			return fmt.Errorf("unknown plugin %s", pluginName)
		}
	}

	pipelineName := viper.GetString("pipeline")
	if len(pipelineName) == 0 {
		return errors.New("pipeline not set")
	}
	pipeline := viper.GetStringSlice("pipelines." + pipelineName)
	if len(pipeline) == 0 {
		return fmt.Errorf("%s pipeline is missing or empty", pipelineName)
	}

	var err error
	if agent.pipeline, err = pipelines.NewPipeline(pipelineName, pipeline, agent.plugins); err != nil {
		return err
	}
	log.Printf("configured %s pipeline", pipelineName)

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

	ticker := time.NewTicker(time.Duration(agent.Interval) * time.Second)

	go func(ctx context.Context) {
		log.Print("agent started")

		for _, plugin := range agent.plugins {
			log.Printf("starting plugin %s", plugin.String())
			if err := plugin.Start(); err != nil {
				log.Printf("failed to start plugin %s: %s", plugin.String(), err)
			}
		}

		tick := func() {
			if err := agent.pipeline.Execute(); err != nil {
				log.Printf("pipeline execute failed: %s", err)
			}
		}

		// Run once at the start before the ticker fires.
		tick()

		for {
			select {
			case <-ctx.Done():
				for _, plugin := range agent.plugins {
					log.Printf("stopping plugin %s", plugin.String())
					plugin.Stop()
				}
				exitCh <- struct{}{}
				return
			case <-ticker.C:
				tick()
			}
		}
	}(cwc)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	go func() {
		select {
		case <-signalCh:
			log.Print("stopping agent ...")
			ticker.Stop()
			cancel()
			return
		}
	}()
	<-exitCh
}
