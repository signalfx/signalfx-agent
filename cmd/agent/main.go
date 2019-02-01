package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
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

var defaultConfigPath = getDefaultConfigPath()

func init() {
	log.SetFormatter(&prefixed.TextFormatter{})
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stdout)
}

func getDefaultConfigPath() string {
	if runtime.GOOS == "windows" {
		exePath, err := os.Executable()
		if err != nil {
			panic("Cannot determine agent executable path, cannot continue")
		}
		configPath, err := filepath.Abs(filepath.Join(filepath.Dir(exePath), "..", "etc", "signalfx", "agent.yaml"))
		if err != nil {
			panic("Cannot determine absolute path of executable parent dir " + exePath)
		}
		return configPath
	}
	return "/etc/signalfx/agent.yaml"
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

// flags is used to store parsed flag values
type flags struct {
	// version is a bool flag for printing the agent version string
	version bool
	// configPath is a string flag for specifying the agent.yaml config file
	configPath string
	// debug is a bool flag for printing debug level information
	debug bool
	// service is a string flag used for starting, stopping, installing or uninstalling the agent as a windows service (windows only)
	service string
	// logEvents is a bool flag for logging events to the Windows Application Event log.
	// This flag is only intended to be used when the agent is launched as a Windows Service.
	logEvents bool
}

// getFlags retrieves flags passed to the agent at runtime and return them in a flags struct
func getFlags() *flags {
	flags := &flags{}
	set := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	set.BoolVar(&flags.version, "version", false, "print agent version")
	set.StringVar(&flags.configPath, "config", defaultConfigPath, "agent config path")
	set.BoolVar(&flags.debug, "debug", false, "print debugging output")

	// service is a windows only feature and should only be added to the flag set on windows
	if runtime.GOOS == "windows" {
		set.StringVar(&flags.service, "service", "", "'start', 'stop', 'install' or 'uninstall' agent as a windows service.  You may specify an alternate config file path with the -config flag when installing the service.")
		set.BoolVar(&flags.logEvents, "logEvents", false, "copy log events from the agent to the Windows Application Event Log.  This is only used when the agent is deployed as a Windows service.  The agent will write to stdout under all other deployment scenarios.")
	}

	// The set is configured to exit on errors so we don't need to check the
	// return value here.
	set.Parse(os.Args[1:])
	if len(set.Args()) > 0 {
		os.Stderr.WriteString("Non-flag parameters are not accepted\n")
		set.Usage()
		os.Exit(2)
	}
	fixGlogFlags()
	return flags
}

func runAgent(flags *flags, interruptCh chan os.Signal, exit chan struct{}) {
	var shutdown context.CancelFunc
	var shutdownComplete <-chan struct{}
	init := func() {
		log.Info("Starting up agent version " + Version)
		shutdown, shutdownComplete = core.Startup(flags.configPath)
	}

	init()

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

	// set the agent version string
	core.VersionLine = fmt.Sprintf("agent-version: %s, built-time: %s\n",
		Version, BuiltTime)

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

		// fetch the commandline flags passed in at runtime
		flags := getFlags()

		if flags.debug {
			log.SetLevel(log.DebugLevel)
		}

		if flags.version {
			fmt.Printf(core.VersionLine)
			os.Exit(0)
		}

		// set up interrupt channel
		interruptCh := make(chan os.Signal, 1)
		signal.Notify(interruptCh, os.Interrupt)
		signal.Notify(interruptCh, syscall.SIGTERM)

		// On windows we start the agent through the package github.com/kardianos/service.
		// The package provides hooks for installing and managing the agent as a windows service.
		if runtime.GOOS == "windows" {
			runAgentWindows(flags, interruptCh)
		} else {
			// create the exit channel that will block until agent is shutdown
			exitCh := make(chan struct{}, 1)
			runAgent(flags, interruptCh, exitCh)
		}
	}

	os.Exit(0)
}
