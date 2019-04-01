// Package pyrunner holds the logic for managing Python plugins using a
// subprocess running Python.
package pyrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

// RuntimeConfig for Python
type RuntimeConfig struct {
	PythonBinary string
	PythonArgs   []string
	// Envvars in the form "key=value".
	PythonEnv []string
}

// PythonRuntimeCustomizable can be implemented by runners that use MonitorCore
// to provide extra config about the Python runtime to use.
type PythonRuntimeCustomizable interface {
	PythonRuntimeConfig() RuntimeConfig
}

// MonitorCore is the adapter to the Python monitor runner process.  It
// communiates with Python using named pipes.  Each general type of Python
// plugin (e.g. Datadog, collectd, etc.) should get its own generic monitor
// struct that uses this adapter by embedding it.
//
// This will run a single, dedicated Python subprocess that actually runs the
// Python monitoring code.  Getting data/metrics/events out of the Python code
// is the responsibility of modules that embed this MonitorCore, hence there
// are no predefined "datapoint" message types.
type MonitorCore struct {
	ctx     context.Context
	cancel  func()
	handler func(MessageReceiver)

	pythonPkg string
	logger    log.FieldLogger

	// Conditional signal that the goroutine that sends does the configuration
	// request sets when configure has been completed.  configResult will hold
	// the result of that configure call.
	configCond   sync.Cond
	configResult error

	// Flag that should be set atomically to tell the goroutine that manages
	// the subprocess whether the process is supposed to be alive or not.
	shutdownCalled int32
}

// New returns a new uninitialized monitor core
func New(pythonPkg string) *MonitorCore {
	ctx, cancel := context.WithCancel(context.Background())

	return &MonitorCore{
		logger:     log.StandardLogger(),
		ctx:        ctx,
		cancel:     cancel,
		pythonPkg:  pythonPkg,
		configCond: sync.Cond{L: &sync.Mutex{}},
	}

}

// Logger returns the logger that should be used
func (mc *MonitorCore) Logger() log.FieldLogger {
	return mc.logger
}

// DefaultRuntimeConfig returns the runtime config that uses the bundled Python
// runtime.
func (mc *MonitorCore) DefaultRuntimeConfig() *RuntimeConfig {
	// The PYTHONHOME envvar is set in agent core when config is processed.
	env := os.Environ()
	env = append(env, config.BundlePythonHomeEnvvar())

	return &RuntimeConfig{
		PythonBinary: defaultPythonBinaryExecutable(),
		PythonArgs:   defaultPythonBinaryArgs(mc.pythonPkg),
		PythonEnv:    env,
	}
}

// run the python subprocess and block until it returns.  Messages from stderr
// (which is remapped to stdout in the Python process, so any "print"-like
// output from Python) will be logged as error logs in the agent.
func (mc *MonitorCore) run(runtimeConf RuntimeConfig, messages *messageReadWriter, stdin io.Reader, stdout io.Writer) error {
	mc.logger.Info("Starting Python runner child process")

	cmd := exec.CommandContext(mc.ctx, runtimeConf.PythonBinary, runtimeConf.PythonArgs...)
	cmd.SysProcAttr = procAttrs()
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Env = runtimeConf.PythonEnv

	// Stderr is just the normal output from the Python code that isn't
	// specially encoded
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	mc.logger = mc.logger.WithFields(log.Fields{
		"runnerPID": cmd.Process.Pid,
	})

	go func() {
		scanner := utils.ChunkScanner(stderr)
		for scanner.Scan() {
			mc.logger.Error(scanner.Text())
		}
	}()

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

// run the Python subprocess, restarting it if it stops while this monitor is
// still active.
func (mc *MonitorCore) runWithRestart(runtimeConf RuntimeConfig, handler func(MessageReceiver), configBytes []byte) {
	for {
		messages, stdin, stdout, err := makePipes()
		if err != nil {
			mc.logger.WithError(err).Error("Couldn't create pipes for Python subprocess")
			return
		}

		go func() {
			mc.configCond.L.Lock()
			mc.configResult = mc.doConfigure(messages, configBytes)
			mc.configCond.L.Unlock()
			// Tell the initial Configure method call that the subproc is done
			// configuring.
			mc.configCond.Broadcast()

			if err != nil {
				mc.Shutdown()
				mc.logger.WithError(mc.configResult).Error("Could not configure Python plugin")
				return
			}

			handler(messages)
		}()

		err = mc.run(runtimeConf, messages, stdin, stdout)

		stdin.Close()
		stdout.Close()
		messages.Close()

		if mc.ShutdownCalled() {
			return
		}

		mc.logger.WithError(err).Error("Python runner process shutdown unexpectedly, restarting...")
		time.Sleep(2 * time.Second)
	}
}

func makePipes() (*messageReadWriter, io.ReadCloser, io.WriteCloser, error) {
	stdinReader, stdinWriter, err := os.Pipe()
	// If this errors, things are really wrong with the system
	if err != nil {
		return nil, nil, nil, err
	}

	stdoutReader, stdoutWriter, err := os.Pipe()
	// If this errors, things are really wrong with the system
	if err != nil {
		return nil, nil, nil, err
	}

	return &messageReadWriter{
		Reader: stdoutReader,
		Writer: stdinWriter,
	}, stdinReader, stdoutWriter, nil
}

// ConfigureInPython sends the given config to the python subproc and returns
// whether configuration was successful.  This method should only be called
// once for the lifetime of the monitor.  The returned MessageReceiver can be
// used to get datapoints/events out of the Python process, the exact format
// of the data is left up to the users of this core.
func (mc *MonitorCore) ConfigureInPython(config config.MonitorCustomConfig, runtimeConfig *RuntimeConfig, handler func(MessageReceiver)) error {
	if mc.handler != nil {
		panic("ConfigureInPython should only be called once")
	}

	mc.handler = handler
	mc.logger = mc.logger.WithFields(log.Fields{
		"monitorID":   config.MonitorConfigCore().MonitorID,
		"monitorType": config.MonitorConfigCore().Type,
	})

	jsonBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}

	mc.configCond.L.Lock()
	defer mc.configCond.L.Unlock()

	if runtimeConfig == nil {
		runtimeConfig = mc.DefaultRuntimeConfig()
	}

	go mc.runWithRestart(*runtimeConfig, handler, jsonBytes)
	mc.configCond.Wait()

	return mc.configResult
}

func (mc *MonitorCore) doConfigure(messages *messageReadWriter, jsonBytes []byte) error {
	messages.SendMessage(MessageTypeConfigure, jsonBytes)

	result, err := mc.waitForConfigure(messages)
	if err != nil {
		return err
	}

	if result.Error != nil {
		return errors.New(*result.Error)
	}

	return nil
}

func (mc *MonitorCore) waitForConfigure(messages *messageReadWriter) (*configResult, error) {
	for {
		msgType, payloadReader, err := messages.RecvMessage()
		if err != nil {
			return nil, err
		}

		content, err := ioutil.ReadAll(payloadReader)
		if err != nil {
			mc.logger.WithError(err).Error("Could not read message from Python")
		}
		payloadReader = bytes.NewBuffer(content)

		switch msgType {
		case MessageTypeConfigureResult:
			var result configResult
			if err := json.NewDecoder(payloadReader).Decode(&result); err != nil {
				return nil, err
			}
			return &result, nil
		case MessageTypeLog:
			if err := mc.HandleLogMessage(payloadReader); err != nil {
				mc.logger.WithError(err).Error("Could not read log message from Python")
			}
		default:
			return nil, fmt.Errorf("got unexpected message code %d from Python", msgType)
		}
	}
}

// ShutdownCalled returns true if the Shutdown method has been called.
func (mc *MonitorCore) ShutdownCalled() bool {
	return atomic.LoadInt32(&mc.shutdownCalled) > 0
}

// Shutdown the whole Runner child process, not just individual monitors
func (mc *MonitorCore) Shutdown() {
	atomic.StoreInt32(&mc.shutdownCalled, 1)

	mc.cancel()
}
