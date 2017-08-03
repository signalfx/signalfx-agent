package neopy

import (
	"encoding/json"
	"sync"

	"github.com/pebbe/zmq4"
	log "github.com/sirupsen/logrus"
)

const registerSocketPath = "ipc:///tmp/signalfx-register.ipc"

// RegisterResponse should contain all of the plugins the neopy process wants
// to register with neo-agent
type RegisterResponse struct {
	Monitors []string
}

type RegisterQueue struct {
	socket *zmq4.Socket
	mutex  sync.Mutex
}

func NewRegisterQueue() *RegisterQueue {
	registerSock, err := zmq4.NewSocket(zmq4.REQ)
	if err != nil {
		panic("Could not create register queue zmq socket: " + err.Error())
	}

	return &RegisterQueue{
		socket: registerSock,
	}
}

func (rq *RegisterQueue) Start() error {
	if err := rq.socket.Bind(registerSocketPath); err != nil {
		return err
	}
	return nil
}

func (rq *RegisterQueue) SocketPath() string {
	return registerSocketPath
}

func (rq *RegisterQueue) GetMonitorList() []string {
	rq.mutex.Lock()
	defer rq.mutex.Unlock()

	_, err := rq.socket.SendMessage("")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Could not send register request to neopy")
		return nil
	}

	respJson, err := rq.socket.RecvBytes(0)
	log.Infof("Received register response %s", respJson)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Failed getting response for register request to NeoPy")
		return nil
	}

	var resp RegisterResponse
	err = json.Unmarshal(respJson, &resp)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Could not parse NeoPy response for register request")
		return nil
	}

	return resp.Monitors
}
