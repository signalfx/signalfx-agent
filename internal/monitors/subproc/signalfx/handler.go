package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mailru/easyjson"
	"github.com/signalfx/com_signalfx_metrics_protobuf"
	"github.com/signalfx/gateway/protocol/signalfx"
	signalfxformat "github.com/signalfx/gateway/protocol/signalfx/format"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/monitors/subproc"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/sirupsen/logrus"
)

const messageTypeDatapointList subproc.MessageType = 200

type JSONHandler struct {
	Output types.Output
	Logger logrus.FieldLogger
}

var _ subproc.MessageHandler = &JSONHandler{}

func (h *JSONHandler) ProcessMessages(ctx context.Context, dataReader subproc.MessageReceiver) {
	for {
		h.Logger.Debug("Waiting for messages")
		msgType, payloadReader, err := dataReader.RecvMessage()

		// This means we are shutdown.
		if ctx.Err() != nil {
			return
		}

		h.Logger.Debugf("Got message of type %d", msgType)

		// This is usually due to the pipe being closed
		if err != nil {
			h.Logger.WithError(err).Error("Could not receive messages")
			return
		}

		if err := h.handleMessage(msgType, payloadReader); err != nil {
			h.Logger.WithError(err).Error("Could not handle message from Python")
			continue
		}
	}
}

func (h *JSONHandler) handleMessage(msgType subproc.MessageType, payloadReader io.Reader) error {
	switch msgType {
	case messageTypeDatapointList:
		// The following is copied from github.com/signalfx/gateway
		var d signalfxformat.JSONDatapointV2
		if err := easyjson.UnmarshalFromReader(payloadReader, &d); err != nil {
			return err
		}
		dps := make([]*datapoint.Datapoint, 0, len(d))
		for metricType, datapoints := range d {
			if len(datapoints) > 0 {
				mt, ok := com_signalfx_metrics_protobuf.MetricType_value[strings.ToUpper(metricType)]
				if !ok {
					h.Logger.Error("Unknown metric type")
					continue
				}
				for _, jsonDatapoint := range datapoints {
					v, err := signalfx.ValueToValue(jsonDatapoint.Value)
					if err != nil {
						h.Logger.WithError(err).Error("Unable to get value for datapoint")
						continue
					}
					dp := datapoint.New(jsonDatapoint.Metric, jsonDatapoint.Dimensions, v, fromMT(com_signalfx_metrics_protobuf.MetricType(mt)), fromTs(jsonDatapoint.Timestamp))
					dps = append(dps, dp)
				}
			}
		}
		for i := range dps {
			h.Output.SendDatapoint(dps[i])
		}

	case subproc.MessageTypeLog:
		return h.HandleLogMessage(payloadReader)
	default:
		return fmt.Errorf("unknown message type received %d", msgType)
	}

	return nil
}

// HandleLogMessage just passes through the reader and logger to the main JSON
// implementation
func (h *JSONHandler) HandleLogMessage(logReader io.Reader) error {
	return HandleLogMessage(logReader, h.Logger)
}

// Copied from github.com/signalfx/gateway
var fromMTMap = map[com_signalfx_metrics_protobuf.MetricType]datapoint.MetricType{
	com_signalfx_metrics_protobuf.MetricType_CUMULATIVE_COUNTER: datapoint.Counter,
	com_signalfx_metrics_protobuf.MetricType_GAUGE:              datapoint.Gauge,
	com_signalfx_metrics_protobuf.MetricType_COUNTER:            datapoint.Count,
}

func fromMT(mt com_signalfx_metrics_protobuf.MetricType) datapoint.MetricType {
	ret, exists := fromMTMap[mt]
	if exists {
		return ret
	}
	panic(fmt.Sprintf("Unknown metric type: %v\n", mt))
}

func fromTs(ts int64) time.Time {
	if ts > 0 {
		return time.Unix(0, ts*time.Millisecond.Nanoseconds())
	}
	return time.Now().Add(-time.Duration(time.Millisecond.Nanoseconds() * ts))
}

// LogMessage represents the log message that comes back from python
type LogMessage struct {
	Message     string  `json:"message"`
	Level       string  `json:"level"`
	Logger      string  `json:"logger"`
	SourcePath  string  `json:"source_path"`
	LineNumber  int     `json:"lineno"`
	CreatedTime float64 `json:"created"`
}

// HandleLogMessage will decode a log message from the given logReader and log
// it using the provided logger.
func HandleLogMessage(logReader io.Reader, logger logrus.FieldLogger) error {
	var msg LogMessage
	err := json.NewDecoder(logReader).Decode(&msg)
	if err != nil {
		return err
	}

	fields := logrus.Fields{
		"logger":      msg.Logger,
		"sourcePath":  msg.SourcePath,
		"lineno":      msg.LineNumber,
		"createdTime": msg.CreatedTime,
	}

	switch msg.Level {
	case "DEBUG":
		logger.WithFields(fields).Debug(msg.Message)
	case "INFO":
		logger.WithFields(fields).Info(msg.Message)
	case "WARNING":
		logger.WithFields(fields).Warn(msg.Message)
	case "ERROR":
		logger.WithFields(fields).Error(msg.Message)
	case "CRITICAL":
		logger.WithFields(fields).Errorf("CRITICAL: %s", msg.Message)
	default:
		logger.WithFields(fields).Info(msg.Message)
	}

	return nil
}
