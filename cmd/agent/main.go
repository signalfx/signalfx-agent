package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/kardianos/service"
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

// flags is used to store parsed flag values
type flags struct {
	// version is a bool flag for printing the agent version string
	version *bool
	// configPath is a string flag for specifying the agent.yaml config file
	configPath *string
	// debug is a bool flag for printing debug level information
	debug *bool
	// service is a string flag used for starting, stopping, installing or uninstalling the agent as a windows service (windows only)
	service *string
}

// getFlags retrieves flags passed to the agent at runtime and return them in a flags struct
func getFlags() *flags {
	flags := &flags{}
	set := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flags.version = set.Bool("version", false, "print agent version")
	flags.configPath = set.String("config", defaultConfigPath, "agent config path")
	flags.debug = set.Bool("debug", false, "print debugging output")

	// service is a windows only feature and should only be added to the flag set on windows
	if runtime.GOOS == "windows" {
		flags.service = set.String("service", "", "'start', 'stop', 'install' or 'uninstall' agent as a windows service.  You may specify an alternate config file path with the -config flag when installing the service.")
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
		shutdown, shutdownComplete = core.Startup(*flags.configPath)
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

type program struct {
	interruptCh chan os.Signal
	exitCh      chan struct{}
	flags       *flags
}

func (p *program) Start(s service.Service) error {
	// create the exit channel that Stop() will block on until agent is shutdown
	p.exitCh = make(chan struct{}, 1)
	go runAgent(p.flags, p.interruptCh, p.exitCh)
	return nil
}

func (p *program) Stop(s service.Service) error {
	// send a signal to shut down the agent
	p.interruptCh <- os.Interrupt
	// wait for the agent to shutdown
	<-p.exitCh
	return nil
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

		// set up interrupt channel
		interruptCh := make(chan os.Signal, 1)
		signal.Notify(interruptCh, os.Interrupt)
		signal.Notify(interruptCh, syscall.SIGTERM)

		// fetch the commandline flags passed in at runtime
		flags := getFlags()

		if *flags.debug {
			log.SetLevel(log.DebugLevel)
		}

		if *flags.version {
			fmt.Printf(core.VersionLine)
			os.Exit(0)
		}

		// we only use github.com/kardianos/service on windows
		if runtime.GOOS == "windows" {
			config := &service.Config{
				Name:        "SignalFx Smart Agent",
				DisplayName: "SignalFx Smart Agent",
				Description: "Collects and publishes metric data to SignalFx",
				Arguments:   []string{"-config", *flags.configPath},
			}
			prgm := &program{
				interruptCh: interruptCh,
				flags:       flags,
			}

			// create the windows service struct
			svc, err := service.New(prgm, config)
			if err != nil {
				log.WithError(err).Error("Failed to find or create the service")
			}

			if *flags.service != "" {
				// install, uninstall, stop the service
				err = service.Control(svc, *flags.service)
			} else {
				// the service library's "start" target does not actually invoke program.Start()
				// we must manually invoke svc.Run() which will then invoke program.Start()
				// svc.Run() will block and run the agent even when the agent is not installed as a service
				err = svc.Run()
			}
			if err != nil {
				log.WithError(err).Error("Failed to control the service")
			}
		} else {
			// create the exit channel that will block until agent is shutdown
			exitCh := make(chan struct{}, 1)
			runAgent(flags, interruptCh, exitCh)
		}
	}

	os.Exit(0)
}
