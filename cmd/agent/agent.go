package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/signalfx/neo-agent/plugins/monitors"
	"github.com/signalfx/neo-agent/plugins/monitors/collectd"
	"github.com/signalfx/neo-agent/plugins/observers"
	"github.com/signalfx/neo-agent/plugins/observers/docker"
	"github.com/signalfx/neo-agent/services"
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
	// Observers used to discover services
	Observers []observers.Observer
	// Monitors that collect metrics
	Monitors []monitors.Monitor
}

// NewAgent with defaults
func NewAgent() *Agent {
	return &Agent{DefaultInterval, make([]observers.Observer, 0), make([]monitors.Monitor, 0)}
}

// Configure an agent using configuration file
func (agent *Agent) Configure(configfile string) error {

	viper.SetDefault("interval", DefaultInterval)

	viper.SetConfigFile(configfile)
	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	agent.Interval = viper.GetInt("interval")

	observer := viper.GetString("observer.name")
	observerConfig := viper.GetStringMapString("observer.configuration")
	switch observer {
	case observers.Docker:
		dockerObserver := docker.NewDocker(observerConfig)
		agent.Observers = append(agent.Observers, dockerObserver)
	}

	monitor := viper.GetString("monitor.name")
	monitorConfig := viper.GetStringMapString("monitor.configuration")
	switch monitor {
	case monitors.Collectd:
		collectdMonitor := collectd.NewCollectd(monitorConfig)
		agent.Monitors = append(agent.Monitors, collectdMonitor)
	}

	return nil
}

// Discover running services by the configured observers
func (agent *Agent) Discover() (map[observers.Observer]services.ServiceInstances, error) {
	result := make(map[observers.Observer]services.ServiceInstances)
	for _, observers := range agent.Observers {
		serviceInstances, err := observers.Discover()
		if err != nil {
			return nil, err
		}
		result[observers] = serviceInstances
	}
	return result, nil
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
		fmt.Printf("failed to configure agent - %s", err)
		os.Exit(1)
	}

	exitCh := make(chan struct{})
	go func(ctx context.Context) {

		fmt.Printf("agent started\n")

		for _, mon := range agent.Monitors {
			log.Printf("starting monitor %s", mon.String())
			if err := mon.Start(); err != nil {
				log.Printf("failed to start monitor %s", mon.String())
			}
		}

		for {
			time.Sleep(time.Duration(agent.Interval) * time.Second)

			result, err := agent.Discover()
			if err != nil {
				log.Printf("failed to discover services- %s", err)
				continue
			}

			for _, v := range result {
				for _, mon := range agent.Monitors {
					// TODO - send cloned observer results to each monitor
					if err := mon.Monitor(v); err != nil {
						log.Printf("failed to monitor observed services - %s", err)
					}
				}
			}

			select {
			case <-ctx.Done():
				for _, mon := range agent.Monitors {
					log.Printf("stopping monitor %s", mon.String())
					mon.Stop()
				}
				exitCh <- struct{}{}
				return
			default:
			}
		}
	}(cwc)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	go func() {
		select {
		case <-signalCh:
			fmt.Println(" stopping agent ..")
			cancel()
			return
		}
	}()
	<-exitCh
}
