package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/signalfx/neo-agent/config"
	"github.com/signalfx/neo-agent/pipelines"
	"github.com/signalfx/neo-agent/plugins"
	_ "github.com/signalfx/neo-agent/plugins/all"
	"github.com/signalfx/neo-agent/watchers"
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

// Agent for monitoring host/service metrics
type Agent struct {
	// Interval to observer service activity
	Interval int
	plugins  []plugins.IPlugin
	pipeline *pipelines.Pipeline
	// configMutex for locking during async config reloads
	configMutex sync.Mutex
	configfile  string
}

// NewAgent with defaults
func NewAgent(configfile string) (*Agent, error) {
	return &Agent{Interval: config.DefaultInterval, configfile: configfile}, nil
}

// Configure an agent by populating the viper config and loading plugins
func (agent *Agent) Configure(changed []string) error {
	agent.configMutex.Lock()
	defer agent.configMutex.Unlock()

	log.Printf("reconfiguring due to changed files: %v", changed)

	if err := config.Load(agent.configfile); err != nil {
		// This is likely a significant error (can't read one or more
		// configuration files) so we don't want to proceed.
		return err
	}

	pluginList, err := plugins.Load(agent.plugins, &agent.configMutex)
	if err == nil {
		log.Printf("replacing plugin set %v with %v", agent.plugins, pluginList)
		agent.plugins = pluginList
		// Reset pipeline which has a reference to the current plugin set.
		log.Println("resetting pipeline")
		agent.pipeline = nil
	} else {
		// If an error is returned then plugin set has not been modified and new
		// plugins have not been started that might be unreference by the plugin
		// set.
		log.Printf("plugin load failed: %s", err)
	}

	agent.Interval = viper.GetInt("interval")

	pipelineName := viper.GetString("pipeline")
	if len(pipelineName) == 0 {
		return errors.New("pipeline not set")
	}
	pipelineConfig := viper.GetStringSlice("pipelines." + pipelineName)
	if len(pipelineConfig) == 0 {
		return fmt.Errorf("%s pipeline is missing or empty", pipelineName)
	}

	agent.pipeline, err = pipelines.NewPipeline(pipelineName, pipelineConfig, agent.plugins)
	if err != nil {
		return fmt.Errorf("failed creating pipeline: %s", err)
	}
	log.Printf("configured %s pipeline", pipelineName)

	return nil
}

func main() {
	var agentConfig = flag.String("config", "/etc/signalfx/agent.yaml", "agent config file")
	var version = flag.Bool("version", false, "print agent version")
	var noWatch = flag.Bool("no-watch", false, "disable watch for changes")

	flag.Parse()

	watch := !*noWatch

	if *version {
		fmt.Printf("agent-version: %s, collectd-version: %s, built-time: %s\n", Version, CollectdVersion, BuiltTime)
		os.Exit(0)
	}

	cwc, cancel := context.WithCancel(context.Background())

	agent, err := NewAgent(*agentConfig)
	if err != nil {
		log.Printf("failed creating agent: %s", err)
		os.Exit(1)
	}

	// We load here to get the polling interval. We need this value to create
	// the watcher. We will reload after the watcher is setup in Configure().
	// (Otherwise changes could be missed.)
	if err := config.Load(agent.configfile); err != nil {
		log.Printf("failed loading config: %s", err)
		os.Exit(1)
	}

	var fsWatcher *watchers.PollingWatcher

	if watch {
		pollingInterval := viper.GetFloat64("pollinginterval")
		if pollingInterval <= 0 {
			log.Printf("pollingInterval must greater than zero")
			os.Exit(1)
		}

		duration := time.Duration(pollingInterval * float64(time.Second))
		fsWatcher = watchers.NewPollingWatcher(agent.Configure, duration)
		log.Printf("watching for changes every %f seconds", duration.Seconds())
		config.WatchForChanges(fsWatcher, *agentConfig)
	}

	if err := agent.Configure(nil); err != nil {
		log.Printf("failed to configure agent: %s", err)
		if !watch {
			// If config reloading is enabled then a configuration change can
			// fix these issues. Otherwise just exit.
			os.Exit(1)
		}
	}

	exitCh := make(chan struct{})

	ticker := time.NewTicker(time.Duration(agent.Interval) * time.Second)

	go func(ctx context.Context) {
		log.Print("agent started")

		tick := func() {
			// Acquire lock so plugins aren't reloaded during execution.
			agent.configMutex.Lock()
			defer agent.configMutex.Unlock()

			if agent.pipeline == nil {
				return
			}

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
			if fsWatcher != nil {
				fsWatcher.Close()
			}
			cancel()
			return
		}
	}()
	<-exitCh
}
