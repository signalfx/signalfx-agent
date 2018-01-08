// +build ignore

package neopy

import (
	"encoding/json"
	"sync"

	"github.com/pebbe/zmq4"
	log "github.com/sirupsen/logrus"
)

const registerSocketPath = "ipc:///tmp/signalfx-register.ipc"

// RegisterResponse should contain all of the monitor types that the neopy
// process wants to register with neo-agent
type RegisterResponse struct {
	Monitors []string
}

// RegisterQueue wraps and manages the zmq socket for doing monitor type
// registration.  This socket is meant to be used as a basic REQ/REP pair, with
// the neo-agent doing the request to NeoPy for its list of moinitor types.
type RegisterQueue struct {
	socket *zmq4.Socket
	mutex  sync.Mutex
}

func newRegisterQueue() *RegisterQueue {
	registerSock, err := zmq4.NewSocket(zmq4.REQ)
	if err != nil {
		panic("Could not create register queue zmq socket: " + err.Error())
	}

	return &RegisterQueue{
		socket: registerSock,
	}
}

func (rq *RegisterQueue) start() error {
	return rq.socket.Bind(registerSocketPath)
}

func (rq *RegisterQueue) socketPath() string {
	return registerSocketPath
}

// getMonitorTypeList queries NeoPy for a list of monitors that it knows about
func (rq *RegisterQueue) getMonitorTypeList() []string {
	rq.mutex.Lock()
	defer rq.mutex.Unlock()

	_, err := rq.socket.SendMessage("")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Could not send register request to neopy")
		return nil
	}

	respJSON, err := rq.socket.RecvBytes(0)
	log.Infof("Received register response %s", respJSON)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Failed getting response for register request to NeoPy")
		return nil
	}

	var resp RegisterResponse
	err = json.Unmarshal(respJSON, &resp)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Could not parse NeoPy response for register request")
		return nil
	}

	return resp.Monitors
}
