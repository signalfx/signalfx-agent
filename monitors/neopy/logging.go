package neopy

import (
	"encoding/json"
	"sync"

	"github.com/pebbe/zmq4"
	log "github.com/sirupsen/logrus"
)

const loggingSocketPath = "ipc:///tmp/signalfx-logs.ipc"
const loggingTopic = "logs"

// LogMessage represents the log message that comes back from python
type LogMessage struct {
	Message     string  `json:"message"`
	Level       string  `json:"level"`
	Logger      string  `json:"logger"`
	SourcePath  string  `json:"source_path"`
	LineNumber  string  `json:"lineno"`
	CreatedTime float64 `json:"created"`
}

// LoggingQueue wraps the zmq socket used to get log messages back from the
// python runner.
type LoggingQueue struct {
	socket *zmq4.Socket
	mutex  sync.Mutex
}

func newLoggingQueue() *LoggingQueue {
	subSock, err := zmq4.NewSocket(zmq4.SUB)
	if err != nil {
		panic("Could not create logging zmq socket: " + err.Error())
	}

	return &LoggingQueue{
		socket: subSock,
	}
}

func (lq *LoggingQueue) start() error {
	if err := lq.socket.Bind(lq.socketPath()); err != nil {
		return err
	}
	return lq.socket.SetSubscribe(loggingTopic)
}

func (lq *LoggingQueue) socketPath() string {
	return loggingSocketPath
}

func (lq *LoggingQueue) listenForLogMessages() {
	go func() {
		for {
			message, err := lq.socket.RecvMessageBytes(0)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("Failed getting log message from NeoPy")
				continue
			}

			var msg LogMessage
			// message[0] is just the topic name
			err = json.Unmarshal(message[1], &msg)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("Could not deserialize log message from NeoPy")
				continue
			}

			lq.handleLogMessage(&msg)
		}
	}()
}

func (lq *LoggingQueue) handleLogMessage(msg *LogMessage) {
	fields := log.Fields{
		"logger":      msg.Logger,
		"sourcePath":  msg.SourcePath,
		"lineno":      msg.LineNumber,
		"createdTime": msg.CreatedTime,
	}

	switch msg.Level {
	case "DEBUG":
		log.WithFields(fields).Debug(msg.Message)
	case "INFO":
		log.WithFields(fields).Info(msg.Message)
	case "WARNING":
		log.WithFields(fields).Warn(msg.Message)
	case "ERROR":
		log.WithFields(fields).Error(msg.Message)
	case "CRITICAL":
		// This will actually kill the agent, perhaps just log at error level
		// instead?
		log.WithFields(fields).Fatal(msg.Message)
	default:
		log.WithFields(fields).Errorf("No log level set for message: %s", msg.Message)
	}
}
