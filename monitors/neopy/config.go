package neopy

import (
	"encoding/json"
	"sync"

	"github.com/pebbe/zmq4"
	log "github.com/sirupsen/logrus"
)

const configureSocketPath = "ipc:///tmp/signalfx-configure.ipc"

// ConfigureResponse is what we expect to be sent by NeoPy to indicate whether
// configuration was successful or not.
type ConfigureResponse struct {
	Success bool
	Error   string
}

// ConfigQueue wraps and manages the zmq REQ socket that sends configuration of
// monitors to NeoPy.
type ConfigQueue struct {
	socket *zmq4.Socket
	mutex  sync.Mutex
}

func newConfigQueue() *ConfigQueue {
	configureSock, err := zmq4.NewSocket(zmq4.REQ)
	if err != nil {
		panic("Couldn't create zmq socket")
	}

	return &ConfigQueue{
		socket: configureSock,
	}
}

func (cq *ConfigQueue) start() error {
	return cq.socket.Bind(cq.socketPath())
}

func (cq *ConfigQueue) socketPath() string {
	return configureSocketPath
}

func (cq *ConfigQueue) configure(conf interface{}) bool {
	cq.mutex.Lock()
	defer cq.mutex.Unlock()

	confJSON, err := json.Marshal(conf)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"config": conf,
		}).Error("Could not serialize monitor config to JSON")
		return false
	}

	_, err = cq.socket.SendMessage(confJSON)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"confJSON": confJSON,
		}).Error("Could not send configuration to neopy")
		return false
	}

	respJSON, err := cq.socket.RecvBytes(0)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"conf":  conf,
		}).Error("Failed getting response when configuring NeoPy")
		return false
	}

	var resp ConfigureResponse
	err = json.Unmarshal(respJSON, &resp)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"conf":  conf,
		}).Error("Could not parse NeoPy response for configure request")
		return false
	}

	if !resp.Success {
		log.WithFields(log.Fields{
			"error":  resp.Error,
			"config": conf,
		}).Error("Failed configuring Python monitor")
		return false
	}

	return true
}
