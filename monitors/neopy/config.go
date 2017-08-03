package neopy

import (
	"encoding/json"
	"sync"

	"github.com/pebbe/zmq4"
	log "github.com/sirupsen/logrus"
)

const configureSocketPath = "ipc:///tmp/signalfx-configure.ipc"

type ConfigureResponse struct {
	Success bool
	Error   string
}

type ConfigQueue struct {
	socket *zmq4.Socket
	mutex  sync.Mutex
}

func NewConfigQueue() *ConfigQueue {
	configureSock, err := zmq4.NewSocket(zmq4.REQ)
	if err != nil {
		panic("Couldn't create zmq socket")
	}

	return &ConfigQueue{
		socket: configureSock,
	}
}

func (cq *ConfigQueue) Start() error {
	if err := cq.socket.Bind(configureSocketPath); err != nil {
		return err
	}
	return nil
}

func (cq *ConfigQueue) SocketPath() string {
	return configureSocketPath
}

func (cq *ConfigQueue) Configure(conf interface{}) bool {
	cq.mutex.Lock()
	defer cq.mutex.Unlock()

	confJson, err := json.Marshal(conf)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"config": conf,
		}).Error("Could not serialize monitor config to JSON")
		return false
	}

	_, err = cq.socket.SendMessage(confJson)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"confJson": confJson,
		}).Error("Could not send configuration to neopy")
		return false
	}

	respJson, err := cq.socket.RecvBytes(0)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"conf":  conf,
		}).Error("Failed getting response when configuring NeoPy")
		return false
	}

	var resp ConfigureResponse
	err = json.Unmarshal(respJson, &resp)
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
