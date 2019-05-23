package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/signalfx/signalfx-agent/internal/core"
	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/selfdescribe"

	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func init() {
	log.SetFormatter(&prefixed.TextFormatter{})
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stdout)
}

const windowsOS = "windows"

func getDefaultConfigPath() string {
	if runtime.GOOS == windowsOS {
		return "\\ProgramData\\SignalFxAgent\\agent.yaml"
	}
	return "/etc/signalfx/agent.yaml"
}

// Set an envvar with the agent's version so that plugins can have easy access
// to it (e.g. metadata plugin).
func setVersionEnvvar() {
	os.Setenv(constants.AgentVersionEnvVar, constants.Version)
}

// Set an envvar with the collectd version so that plugins have easy access to it
func setCollectdVersionEnvvar() {
	os.Setenv(constants.CollectdVersionEnvVar, constants.CollectdVersion)
}

// Print out status about an existing instance of the agent.
func doStatus() {
	set := flag.NewFlagSet("status", flag.ExitOnError)
	configPath := set.String("config", getDefaultConfigPath(), "agent config path")
	set.Usage = func() {
		fmt.Fprintf(set.Output(), "Usage: signalfx-agent status [all | monitors | config | endpoints]\n\n"+
			"  The optional section arg can be one of the following:\n"+
			"    all - Dump everything available\n"+
			"    monitors - Show information about all active monitors\n"+
			"    config - Show the fully resolved configuration currently in use by the agent\n"+
			"    endpoints - Show the discovered endpoints available to discovery rules\n"+
			"  If no section arg is provided, a short summary of the agent is output\n\n")
		set.PrintDefaults()
	}

	_ = set.Parse(os.Args[2:])

	log.SetLevel(log.ErrorLevel)

	var section string
	if len(set.Args()) == 1 {
		section = set.Args()[0]
	} else if len(set.Args()) > 1 {
		set.Usage()
		os.Exit(4)
	}

	status, err := core.Status(*configPath, section)
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
	selfdescribe.JSON(os.Stdout)
}

var dpTapUsage = `
If no filters are specified, all datapoints will be output.

Examples:

  Get all metrics that start with 'ps_' that have a plugin_instance dimension that starts with 'java':

    signalfx-agent tap-dps -metric 'ps_*' -dims '{plugin_instance: java*}'

`

func doDatapointTap() {
	set := flag.NewFlagSet("tap-dps", flag.ExitOnError)
	set.Usage = func() {
		fmt.Fprintf(set.Output(), "Usage of %s tap-dps:\n", os.Args[0])
		set.PrintDefaults()
		fmt.Fprint(set.Output(), dpTapUsage)
	}

	configPath := set.String("config", getDefaultConfigPath(), "agent config path")
	metric := set.String("metric", "", "metric name filter string -- accepts globs")
	dims := set.String("dims", "", "dimension filter string in compact YAML map notation -- dimension values can be globbed")

	if err := set.Parse(os.Args[2:]); err != nil {
		set.Usage()
		os.Exit(1)
	}

	stream, err := core.StreamDatapoints(*configPath, *metric, *dims)
	if err != nil {
		fmt.Printf("Could not stream datapoints: %v", err)
		return
	}

	_, err = io.Copy(os.Stdout, stream)
	if err != io.EOF && err != nil {
		fmt.Printf("Error streaming datapoints: %v", err)
	}
}

// glog is a transitive dependency of the agent and puts a bunch of flags in
// the flag package.  We don't really ever need to have users override these,
// but we would like ERROR messages going to stderr of the agent instead of to
// a temporary file.
func fixGlogFlags() {
	os.Args = os.Args[:1]
	flag.Parse()
	_ = flag.Set("logtostderr", "true")
}

// flags is used to store parsed flag values
type flags struct {
	// configPath is a string flag for specifying the agent.yaml config file
	configPath string
	// service is a string flag used for starting, stopping, installing or
	// uninstalling the agent as a windows service (windows only)
	service string
	// logEvents is a bool flag for logging events to the Windows Application Event log.
	// This flag is only intended to be used when the agent is launched as a Windows Service.
	logEvents bool
	// version is a bool flag for printing the agent version string
	version bool
	// debug is a bool flag for printing debug level information
	debug bool
}

// getFlags retrieves flags passed to the agent at runtime and return them in a flags struct
func getFlags() *flags {
	flags := &flags{}
	set := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	set.BoolVar(&flags.version, "version", false, "print agent version")
	set.StringVar(&flags.configPath, "config", getDefaultConfigPath(), "agent config path")
	set.BoolVar(&flags.debug, "debug", false, "print debugging output")

	// service is a windows only feature and should only be added to the flag set on windows
	if runtime.GOOS == windowsOS {
		set.StringVar(&flags.service, "service", "", "'start', 'stop', 'install' or 'uninstall' agent as a windows service.  You may specify an alternate config file path with the -config flag when installing the service.")
		set.BoolVar(&flags.logEvents, "logEvents", false, "copy log events from the agent to the Windows Application Event Log.  This is only used when the agent is deployed as a Windows service.  The agent will write to stdout under all other deployment scenarios.")
	}

	// The set is configured to exit on errors so we don't need to check the
	// return value here.
	_ = set.Parse(os.Args[1:])
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
		logrus.Info("Starting up agent version " + constants.Version)
		shutdown, shutdownComplete = core.Startup(flags.configPath)
	}

	init()

	go func() {
		<-interruptCh
		logrus.Info("Interrupt signal received, stopping agent")
		shutdown()
		select {
		case <-shutdownComplete:
			break
		case <-time.After(10 * time.Second):
			logrus.Error("Shutdown timed out, forcing process down")
			break
		}
		close(exit)
	}()

	hupCh := make(chan os.Signal, 1)
	signal.Notify(hupCh, syscall.SIGHUP)
	go func() {
		for range hupCh {
			logrus.Info("Forcing agent reset")
			shutdown()
			<-shutdownComplete
			init()
		}
	}()

	<-exit
}

func main() {
	setVersionEnvvar()
	setCollectdVersionEnvvar()

	// set the agent version string
	core.VersionLine = fmt.Sprintf("agent-version: %s, built-time: %s\n",
		constants.Version, constants.BuildTime)

	// Make it so the symlink from agent-status to this binary invokes the
	// status command
	if strings.HasSuffix(os.Args[0], "agent-status") {
		os.Args = append([]string{os.Args[0], "status"}, os.Args[1:]...)
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
	case "tap-dps":
		doDatapointTap()
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

		// create the exit channel that will block until agent is shutdown
		exitCh := make(chan struct{}, 1)
		// On windows we start the agent through the package github.com/kardianos/service.
		// The package provides hooks for installing and managing the agent as a windows service.
		runAgentPlatformSpecific(flags, interruptCh, exitCh)
	}

	os.Exit(0)
}
