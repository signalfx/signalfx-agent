package logging

import "github.com/sirupsen/logrus"

// NewLogger creates a telegraf.Logger instance wrapper
func NewLogger(log logrus.FieldLogger) *TelegrafLogger {
	return &TelegrafLogger{log: log}
}

type TelegrafLogger struct {
	log logrus.FieldLogger
}

func (t *TelegrafLogger) Errorf(format string, args ...interface{}) {
	t.log.Errorf(format, args...)
}

func (t *TelegrafLogger) Error(args ...interface{}) {
	t.log.Error(args...)
}

func (t *TelegrafLogger) Debugf(format string, args ...interface{}) {
	t.log.Debugf(format, args...)
}

func (t *TelegrafLogger) Debug(args ...interface{}) {
	t.log.Debug(args...)
}

func (t *TelegrafLogger) Warnf(format string, args ...interface{}) {
	t.log.Warnf(format, args...)
}

func (t *TelegrafLogger) Warn(args ...interface{}) {
	t.log.Warn(args...)
}

func (t *TelegrafLogger) Infof(format string, args ...interface{}) {
	t.log.Infof(format, args...)
}

func (t *TelegrafLogger) Info(args ...interface{}) {
	t.log.Info(args...)
}
