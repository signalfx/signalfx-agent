package neopy

import (
	"encoding/json"
	"sync"

	"github.com/pebbe/zmq4"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/neo-agent/core/config/types"
	log "github.com/sirupsen/logrus"
)

const datapointsSocketPath = "ipc:///tmp/signalfx-datapoints.ipc"
const datapointsTopic = "datapoints"

// DatapointMessage represents the message sent by python with a datapoint
type DatapointMessage struct {
	MonitorID types.MonitorID `json:"monitor_id"`
	// Will be deserialized by the golib method by itself
	Datapoint *datapoint.Datapoint
}

// DatapointsQueue wraps the zmq socket used to get datapoints back from python
type DatapointsQueue struct {
	socket *zmq4.Socket
	mutex  sync.Mutex
}

func newDatapointsQueue() *DatapointsQueue {
	subSock, err := zmq4.NewSocket(zmq4.SUB)
	if err != nil {
		panic("Could not create datapoints zmq socket: " + err.Error())
	}

	return &DatapointsQueue{
		socket: subSock,
	}
}

func (dq *DatapointsQueue) start() error {
	if err := dq.socket.Bind(dq.socketPath()); err != nil {
		return err
	}
	if err := dq.socket.SetSubscribe(datapointsTopic); err != nil {
		return err
	}
	return nil
}

func (dq *DatapointsQueue) socketPath() string {
	return datapointsSocketPath
}

func (dq *DatapointsQueue) listenForDatapoints() <-chan *DatapointMessage {
	dq.mutex.Lock()

	ch := make(chan *DatapointMessage)
	go func() {
		defer dq.mutex.Unlock()

		for {
			log.Debug("waiting for datapoints")
			message, err := dq.socket.RecvMessageBytes(0)
			log.Debug("got datapoint")
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("Failed getting datapoints from NeoPy")
				continue
			}

			var msg struct {
				MonitorID types.MonitorID `json:"monitor_id"`
				// Will be deserialized by the golib method by itself
				Datapoint *json.RawMessage
			}
			// message[0] is just the topic name
			err = json.Unmarshal(message[1], &msg)
			if err != nil {
				log.WithFields(log.Fields{
					"error":   err,
					"message": message,
				}).Error("Could not deserialize datapoint message from NeoPy")
				continue
			}

			var dp datapoint.Datapoint
			err = dp.UnmarshalJSON(*msg.Datapoint)
			if err != nil {
				log.WithFields(log.Fields{
					"dpJSON": string(*msg.Datapoint),
					"error":  err,
				}).Error("Could not deserialize datapoint from NeoPy")
				continue
			}

			log.WithFields(log.Fields{
				"msg": msg,
			}).Debug("Datapoint Received from NeoPy")

			ch <- &DatapointMessage{
				MonitorID: msg.MonitorID,
				Datapoint: &dp,
			}
		}
	}()
	return ch
}
