package pyrunner

import (
	"encoding/json"
	"io"

	log "github.com/sirupsen/logrus"
)

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
func (mc *MonitorCore) HandleLogMessage(logReader io.Reader) error {
	var msg LogMessage
	err := json.NewDecoder(logReader).Decode(&msg)
	if err != nil {
		return err
	}

	fields := log.Fields{
		"logger":      msg.Logger,
		"sourcePath":  msg.SourcePath,
		"lineno":      msg.LineNumber,
		"createdTime": msg.CreatedTime,
	}

	switch msg.Level {
	case "DEBUG":
		mc.logger.WithFields(fields).Debug(msg.Message)
	case "INFO":
		mc.logger.WithFields(fields).Info(msg.Message)
	case "WARNING":
		mc.logger.WithFields(fields).Warn(msg.Message)
	case "ERROR":
		mc.logger.WithFields(fields).Error(msg.Message)
	case "CRITICAL":
		mc.logger.WithFields(fields).Errorf("CRITICAL: %s", msg.Message)
	default:
		mc.logger.WithFields(fields).Info(msg.Message)
	}

	return nil
}
