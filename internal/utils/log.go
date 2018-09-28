package utils

import (
	"github.com/signalfx/golib/log"

	"github.com/sirupsen/logrus"
)

// LogrusGolibShim makes a Logrus logger conform to the golib Log interface
type LogrusGolibShim struct {
	logrus.FieldLogger
}

var _ log.Logger = &LogrusGolibShim{}

// Log conforms to the golib Log interface
func (l *LogrusGolibShim) Log(keyvals ...interface{}) {
	fields := logrus.Fields{}

	var currentKey *log.Key
	messages := []interface{}{}

	for k := range keyvals {
		switch v := keyvals[k].(type) {
		case log.Key:
			currentKey = &v
		default:
			if currentKey != nil {
				switch *currentKey {
				case log.Msg:
					messages = append(messages, v)
				default:
					fields[string(*currentKey)] = v
				}
				currentKey = nil
			} else {
				messages = append(messages, v)
			}
		}
	}

	fieldlog := logrus.WithFields(fields)

	if _, ok := fields[string(log.Err)]; ok {
		fieldlog.Error(messages...)
	} else {
		fieldlog.Info(messages...)
	}
}
