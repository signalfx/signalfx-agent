// +build ignore

package neopy

import (
	"encoding/json"
	"sync"

	"github.com/pebbe/zmq4"
	"github.com/signalfx/neo-agent/monitors"
	log "github.com/sirupsen/logrus"
)

const shutdownSocketPath = "ipc:///tmp/signalfx-shutdown.ipc"
const shutdownTopic = "shutdown"

// ShutdownRequest represents the message sent to python to tell it to shutdown
// a monitor
type ShutdownRequest struct {
	MonitorID string `json:"monitor_id"`
}

// ShutdownQueue is used to tell NeoPy to shutdown specific monitors when they
// are no longer configured or have running services
type ShutdownQueue struct {
	socket *zmq4.Socket
	mutex  sync.Mutex
}

func newShutdownQueue() *ShutdownQueue {
	pubSock, err := zmq4.NewSocket(zmq4.PUB)
	if err != nil {
		panic("Could not create shutdown zmq socket: " + err.Error())
	}

	return &ShutdownQueue{
		socket: pubSock,
	}
}

func (sq *ShutdownQueue) start() error {
	return sq.socket.Bind(sq.socketPath())
}

func (sq *ShutdownQueue) socketPath() string {
	return shutdownSocketPath
}

func (sq *ShutdownQueue) sendShutdownForMonitor(monitorID monitors.MonitorID) bool {
	sq.mutex.Lock()
	defer sq.mutex.Unlock()

	reqJSON, err := json.Marshal(ShutdownRequest{
		MonitorID: string(monitorID),
	})
	if err != nil {
		log.WithFields(log.Fields{
			"error":     err,
			"monitorID": monitorID,
		}).Error("Could not serialize shutdown request to JSON")
		return false
	}

	_, err = sq.socket.SendMessage(shutdownTopic, reqJSON)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"reqJson": reqJSON,
		}).Error("Could not send shutdown notice to neopy")
		return false
	}

	log.WithFields(log.Fields{
		"monitorID": monitorID,
	}).Debug("Shutting down NeoPy monitor")

	return true
}
