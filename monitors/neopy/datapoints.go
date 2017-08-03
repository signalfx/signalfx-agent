package neopy

import (
	"encoding/json"
	"sync"

	"github.com/pebbe/zmq4"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/neo-agent/core/config"
	log "github.com/sirupsen/logrus"
)

const datapointsSocketPath = "ipc:///tmp/signalfx-datapoints.ipc"

type DatapointMessage struct {
	MonitorID config.MonitorID `json:"monitor_id"`
	// Will be deserialized by the golib method by itself
	Datapoint *datapoint.Datapoint
}

type DatapointsQueue struct {
	socket *zmq4.Socket
	mutex  sync.Mutex
}

func NewDatapointsQueue() *DatapointsQueue {
	subSock, err := zmq4.NewSocket(zmq4.SUB)
	if err != nil {
		panic("Could not create datapoints zmq socket: " + err.Error())
	}

	return &DatapointsQueue{
		socket: subSock,
	}
}

func (dq *DatapointsQueue) Start() error {
	if err := dq.socket.Bind(datapointsSocketPath); err != nil {
		return err
	}
	if err := dq.socket.SetSubscribe("datapoints"); err != nil {
		return err
	}
	return nil
}

func (mq *DatapointsQueue) SocketPath() string {
	return datapointsSocketPath
}

func (mq *DatapointsQueue) ListenForDatapoints() <-chan *DatapointMessage {
	ch := make(chan *DatapointMessage)
	go func() {
		for {
			log.Debug("waiting for datapoints")
			message, err := mq.socket.RecvMessageBytes(0)
			log.Debug("got datapoint")
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("Failed getting datapoints from NeoPy")
				continue
			}

			var msg struct {
				MonitorID config.MonitorID `json:"monitor_id"`
				// Will be deserialized by the golib method by itself
				Datapoint *json.RawMessage
			}
			// message[0] is just the topic name
			err = json.Unmarshal(message[1], &msg)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
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
