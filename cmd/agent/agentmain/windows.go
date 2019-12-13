// +build windows

package agentmain

import (
	"os"

	"github.com/StackExchange/wmi"
	"github.com/kardianos/service"
	log "github.com/sirupsen/logrus"
)

// WindowsEventLogHook is a logrus log hook for emitting to the Windows Application Events log.
// Events will only be raised when the agent is run as a Windows Service with the "-logEvents" flag enabled
// Log events may be viewed using the "Event Viewer" Windows Administrative Tool.
type WindowsEventLogHook struct {
	logger service.Logger
}

// Fire is a call back for logrus entries to be passed through the hook
func (h *WindowsEventLogHook) Fire(entry *log.Entry) error {
	msg, err := entry.String()
	if err != nil {
		return err
	}

	// Windows Events support other log levels, but the golang.org/x/sys/windows library
	// does not support these, and therefore the github.com/kardianos/service library does not support them.
	// TODO: If our dependencies ever add support for Critical and Verbose log levels,
	// then we should utilize them.
	switch entry.Level {
	case log.PanicLevel:
		return h.logger.Error(msg)
	case log.FatalLevel:
		return h.logger.Error(msg)
	case log.ErrorLevel:
		return h.logger.Error(msg)
	case log.WarnLevel:
		return h.logger.Warning(msg)
	case log.InfoLevel:
		return h.logger.Info(msg)
	case log.DebugLevel:
		return h.logger.Info(msg)
	default:
		return nil
	}
}

// Levels returns the logrus levels that the WindowsEventLogHook handles
func (h *WindowsEventLogHook) Levels() []log.Level {
	return log.AllLevels
}

// program is a struct that is used by service.Service to control the agent
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
	p.interruptCh <- os.Kill
	// wait for the agent to shutdown
	<-p.exitCh
	return nil
}

// runAgentPlatformSpecific is responsible for wrapping the agent in a windows service structure.
// This structure is used even when the agent is not registered as a service.
// The original runAgent function is invoked as part of the method program.Start() which itself
// is invoked by service.Service.Run()
func runAgentPlatformSpecific(flags *flags, interruptCh chan os.Signal, exitCh chan struct{}) {
	initializeWMI()

	config := &service.Config{
		Name:        "signalfx-agent",
		DisplayName: "SignalFx Smart Agent",
		Description: "Collects and publishes metric data to SignalFx",
		Arguments:   []string{"-config", flags.configPath},
	}

	// add the logEvents argument to the service to enable Windows Application Event logging
	if flags.logEvents {
		config.Arguments = append(config.Arguments, "-logEvents")
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

	// setup Windows Event Logging.  The logging hook will only work when the agent
	// is deployed as a service with the "-logEvents" flag.
	if flags.logEvents {
		logger, err := svc.Logger(make(chan error, 500))
		if err != nil {
			log.WithError(err).Error("Unable to set up windows event logger")
		}
		log.AddHook(&WindowsEventLogHook{logger: logger})
	}

	if flags.service != "" {
		// install, uninstall, stop the service
		err = service.Control(svc, flags.service)
	} else {
		// the service library's "start" target does not actually invoke program.Start()
		// we must manually invoke svc.Run() which will then invoke program.Start()
		// svc.Run() will block and run the agent even when the agent is not installed as a service
		err = svc.Run()
	}

	if err != nil {
		log.WithError(err).Error("Failed to control the service")
	}
}

// See https://github.com/StackExchange/wmi/issues/27#issuecomment-309578576.
// This might prevent memory leaks.
func initializeWMI() {
	log.Info("Initializing WMI SWbemServices")
	s, err := wmi.InitializeSWbemServices(wmi.DefaultClient)
	if err != nil {
		log.WithError(err).Error("Could not initialize WMI properly")
		return
	}
	wmi.DefaultClient.SWbemServicesClient = s
}
