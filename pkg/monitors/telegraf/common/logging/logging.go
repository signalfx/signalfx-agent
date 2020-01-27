package logging

import "github.com/sirupsen/logrus"

// NewLogger creates a telegraf.Logger instance wrapper
func NewLogger(log logrus.FieldLogger) *telegrafLogger {
	return &telegrafLogger{log: log}
}

type telegrafLogger struct {
	log logrus.FieldLogger
}

func (t *telegrafLogger) Errorf(format string, args ...interface{}) {
	t.log.Errorf(format, args...)
}

func (t *telegrafLogger) Error(args ...interface{}) {
	t.log.Error(args...)
}

func (t *telegrafLogger) Debugf(format string, args ...interface{}) {
	t.log.Debugf(format, args...)
}

func (t *telegrafLogger) Debug(args ...interface{}) {
	t.log.Debug(args...)
}

func (t *telegrafLogger) Warnf(format string, args ...interface{}) {
	t.log.Warnf(format, args...)
}

func (t *telegrafLogger) Warn(args ...interface{}) {
	t.log.Warn(args...)
}

func (t *telegrafLogger) Infof(format string, args ...interface{}) {
	t.log.Infof(format, args...)
}

func (t *telegrafLogger) Info(args ...interface{}) {
	t.log.Info(args...)
}
