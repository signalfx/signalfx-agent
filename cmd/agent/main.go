package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/signalfx-agent/internal/core"
	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/selfdescribe"

	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var (
	// Version for agent
	Version string

	// CollectdVersion for collectd
	CollectdVersion string

	// BuiltTime for the agent
	BuiltTime string
)

const defaultConfigPath = "/etc/signalfx/agent.yaml"

func init() {
	log.SetFormatter(&prefixed.TextFormatter{})
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stdout)
}

// Set an envvar with the agent's version so that plugins can have easy access
// to it (e.g. metadata plugin).
func setVersionEnvvar() {
	os.Setenv(constants.AgentVersionEnvVar, Version)
}

// Set an envvar with the collectd version so that plugins have easy access to it
func setCollectdVersionEnvvar() {
	os.Setenv(constants.CollectdVersionEnvVar, CollectdVersion)
}

// Print out status about an existing instance of the agent.
func doStatus() {
	set := flag.NewFlagSet("status", flag.ExitOnError)
	configPath := set.String("config", defaultConfigPath, "agent config path")

	set.Parse(os.Args[2:])

	log.SetLevel(log.ErrorLevel)

	status, err := core.Status(*configPath)
	if err != nil {
		fmt.Printf("Could not get status: %s\nAre you sure the agent is currently running?\n", err)
		os.Exit(1)
	}
	fmt.Print(string(status))
	fmt.Println("")
}

// Print out agent self-description of config/metadata
func doSelfDescribe() {
	log.SetOutput(os.Stderr)
	fmt.Print(selfdescribe.JSON())
}

// glog is a transitive dependency of the agent and puts a bunch of flags in
// the flag package.  We don't really ever need to have users override these,
// but we would like ERROR messages going to stderr of the agent instead of to
// a temporary file.
func fixGlogFlags() {
	os.Args = os.Args[:1]
	flag.Parse()
	flag.Set("logtostderr", "true")
}

func runAgent() {
	set := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	version := set.Bool("version", false, "print agent version")
	configPath := set.String("config", defaultConfigPath, "agent config path")
	debug := set.Bool("debug", false, "print debugging output")

	core.VersionLine = fmt.Sprintf("agent-version: %s, built-time: %s\n",
		Version, BuiltTime)

	// The set is configured to exit on errors so we don't need to check the
	// return value here.
	set.Parse(os.Args[1:])
	if len(set.Args()) > 0 {
		os.Stderr.WriteString("Non-flag parameters are not accepted\n")
		set.Usage()
		os.Exit(2)
	}
	fixGlogFlags()

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
		log.Info("Starting up agent version " + Version)
		shutdown, shutdownComplete = core.Startup(*configPath)
	}

	init()

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	signal.Notify(interruptCh, syscall.SIGTERM)
	go func() {
		select {
		case <-interruptCh:
			log.Info("Interrupt signal received, stopping agent")
			shutdown()
			select {
			case <-shutdownComplete:
				break
			case <-time.After(10 * time.Second):
				log.Error("Shutdown timed out, forcing process down")
				break
			}

			close(exit)
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

func main() {
	setVersionEnvvar()
	setCollectdVersionEnvvar()

	// Make it so the symlink from agent-status to this binary invokes the
	// status command
	if len(os.Args) == 1 && strings.HasSuffix(os.Args[0], "agent-status") {
		os.Args = append(os.Args, "status")
	}

	var firstArg string
	if len(os.Args) >= 2 {
		firstArg = os.Args[1]
	}

	switch firstArg {
	case "status":
		doStatus()
	case "selfdescribe":
		doSelfDescribe()
	default:
		if firstArg != "" && !strings.HasPrefix(firstArg, "-") {
			log.Errorf("Unknown subcommand '%s'", firstArg)
			os.Exit(127)
		}
		runAgent()
	}

	os.Exit(0)
}
