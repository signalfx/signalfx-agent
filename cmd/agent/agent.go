package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"sync"

	"github.com/signalfx/neo-agent/config"
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
)

// Agent for monitoring host/service metrics
type Agent struct {
	// Interval to observer service activity
	Interval   int
	pipeline   *pipelines.Pipeline
	configfile string
	plugins    plugins.Manager
}

// NewAgent with defaults
func NewAgent(configfile string) (*Agent, error) {
	return &Agent{Interval: config.DefaultInterval, configfile: configfile}, nil
}

// Configure an agent by populating the viper config and loading plugins
func (agent *Agent) Configure() error {
	sourceConfig := viper.Sub("stores")
	if sourceConfig == nil {
		return errors.New("stores config missing")
	}

	// Reconfigure stores.
	if err := config.Stores.Config(sourceConfig); err != nil {
		return err
	}

	pluginList, err := agent.plugins.Load()
	if err == nil {
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

	agent.pipeline, err = pipelines.NewPipeline(pipelineName, pipelineConfig, pluginList)
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

	versionLine := fmt.Sprintf("agent-version: %s, collectd-version: %s, built-time: %s\n",
	                           Version, CollectdVersion, BuiltTime)

	// Override Usage to support the signalfx-metadata plugin, which expects a
	// line with the collectd version from the -h flag.
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, versionLine)
	}

	flag.Parse()

	watch := !*noWatch
	viper.SetDefault("filewatching", watch)

	if *version {
		fmt.Printf(versionLine)
		os.Exit(0)
	}

	cwc, cancel := context.WithCancel(context.Background())

	agent, err := NewAgent(*agentConfig)
	if err != nil {
		log.Printf("failed creating agent: %s", err)
		os.Exit(1)
	}

	reload := make(chan struct{})
	var configMutex sync.Mutex

	// We load here to get the polling interval. We need this value to create
	// the watcher. We will reload after the watcher is setup in Configure().
	// (Otherwise changes could be missed.)
	if err := config.Init(agent.configfile, reload, &configMutex); err != nil {
		log.Printf("failed loading config: %s", err)
		os.Exit(1)
	}

	if err := agent.Configure(); err != nil {
		log.Printf("error configuring agent: %s", err)
	}

	exitCh := make(chan struct{})

	ticker := time.NewTicker(time.Duration(agent.Interval) * time.Second)

	go func(ctx context.Context) {
		log.Print("agent started")

		tick := func() {
			// Acquire lock so viper instance (thread unsafe) isn't modified
			// mid-flight.
			configMutex.Lock()
			defer configMutex.Unlock()

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
			case <-reload:
				log.Print("reconfiguring agent from changed configuration")
				if err := agent.Configure(); err != nil {
					log.Printf("error reconfiguring agent: %s", err)
				}
			case <-ctx.Done():
				agent.plugins.Stop()
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
	log.Print("stopping stores")
	config.Stores.Close()
}
