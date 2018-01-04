// NeoAgent
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
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

// We always want the current dir of the agent to be the base of the bundle.  A
// lot of stuff depends on relative paths from here so that the agent is more
// easily relocated.
// This assumes the agent binary is in the bin dir in the root of the bundle.
func setCurrentDir() {
	exePath, err := os.Executable()
	if err != nil {
		panic("Cannot determine agent executable path, cannot continue")
	}
	os.Chdir(filepath.Join(filepath.Dir(exePath), ".."))
}

// Set an envvar with the agent's version so that plugins can have easy access
// to it (e.g. metadata plugin).
func setVersionEnvvar() {
	os.Setenv("SIGNALFX_AGENT_VERSION", Version)
}

func main() {
	setCurrentDir()
	setVersionEnvvar()

	configPath := flag.String("config", "/etc/signalfx/agent.yaml", "agent config path")
	version := flag.Bool("version", false, "print agent version")
	debug := flag.Bool("debug", false, "print debugging output")

	core.VersionLine = fmt.Sprintf("agent-version: %s, collectd-version: %s, built-time: %s\n",
		Version, CollectdVersion, BuiltTime)

	// Override Usage to support the signalfx-metadata plugin, which expects a
	// line with the collectd version from the -h flag.
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, core.VersionLine)
	}

	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	if *version {
		fmt.Printf(core.VersionLine)
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
			<-shutdownComplete
			exit <- struct{}{}
		}
	}()

	hupCh := make(chan os.Signal, 1)
	signal.Notify(hupCh, syscall.SIGHUP)
	go func() {
		for {
			select {
			case <-hupCh:
				log.Info("Forcing agent reset")
				shutdown()
				<-shutdownComplete
				init()
			}
		}
	}()

	<-exit
}
