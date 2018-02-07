// NeoAgent
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/neo-agent/core"

	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var (
	// Version for agent
	Version string
	// BuiltTime for the agent
	BuiltTime string
)

const defaultConfigPath = "/etc/signalfx/agent.yaml"

func init() {
	log.SetFormatter(&prefixed.TextFormatter{})
	log.SetLevel(log.InfoLevel)
}

// We always want the current dir of the agent to be the base of the bundle.  A
// lot of stuff depends on relative paths from here so that the agent is more
// easily relocated.
// If the SIGNALFX_BUNDLE_DIR envvar is set, this will be used, otherwise this
// assumes the agent binary is in the bin dir in the root of the bundle.
// We also need to set some environment variables that collectd makes use of
// for loading c plugins as well as the Java and Python runtimes.
func setupEnvironment() {
	bundleDir := os.Getenv("SIGNALFX_BUNDLE_DIR")
	if bundleDir == "" {
		exePath, err := os.Executable()
		if err != nil {
			panic("Cannot determine agent executable path, cannot continue")
		}
		bundleDir = filepath.Join(filepath.Dir(exePath), "..")
	}
	os.Chdir(bundleDir)

	os.Setenv("LD_LIBRARY_PATH", filepath.Join(bundleDir, "lib"))
	os.Setenv("JAVA_HOME", filepath.Join(bundleDir, "jvm/java-8-openjdk-amd64"))
	os.Setenv("PYTHONHOME", bundleDir)
}

// Set an envvar with the agent's version so that plugins can have easy access
// to it (e.g. metadata plugin).
func setVersionEnvvar() {
	os.Setenv("SIGNALFX_AGENT_VERSION", Version)
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

// Print out agent documentation
func doDocs() {
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
	watchDuration := set.Duration("filePollRate", 5*time.Second,
		"Set to 0 to disable config file watch, see https://golang.org/pkg/time/#ParseDuration for acceptable values")

	core.VersionLine = fmt.Sprintf("agent-version: %s, built-time: %s\n",
		Version, BuiltTime)

	// The set is configured to exit on errors so we don't need to check the
	// return value here.
	set.Parse(os.Args[1:])
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
		shutdown, shutdownComplete = core.Startup(*configPath, *watchDuration)
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
			<-shutdownComplete
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
	setupEnvironment()
	setVersionEnvvar()

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
	case "docs":
		doDocs()
	default:
		runAgent()
	}

	os.Exit(0)
}
