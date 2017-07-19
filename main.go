package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/neo-agent/core"

	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var (
	// Version for agent
	Version string
	// BuiltTime for the agent
	BuiltTime string
	// CollectdVersion embedded in agent
	CollectdVersion string
)

func init() {
	log.SetFormatter(&prefixed.TextFormatter{})
	log.SetLevel(log.InfoLevel)
}

func main() {
	configPath := flag.String("config", "/etc/signalfx/agent.yaml", "agent config path")
	version := flag.Bool("version", false, "print agent version")
	debug := flag.Bool("debug", false, "print debugging output")

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

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	if *version {
		fmt.Printf(versionLine)
		os.Exit(0)
	}

	exit := make(chan struct{})

	var shutdown context.CancelFunc
	var shutdownComplete <-chan struct{}
	init := func() {
		shutdown, shutdownComplete = core.Startup(*configPath)
	}

	init()

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	go func() {
		select {
		case <-interruptCh:
			log.Info("Interrupt signal received, stopping agent")
			shutdown()
			exit <- struct{}{}
		}
	}()

	hupCh := make(chan os.Signal, 1)
	signal.Notify(hupCh, syscall.SIGHUP)
	go func() {
		select {
		case <-hupCh:
			log.Info("Forcing agent reset")
			shutdown()
			<-shutdownComplete
			init()
		}
	}()

	<-exit
}
