package write

import (
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/davecgh/go-spew/spew"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/golib/log"
	"github.com/signalfx/golib/pointer"
	"github.com/signalfx/metricproxy/protocol/collectd"
	"github.com/sirupsen/logrus"
)

// Server is a wrapper around MetricProxy's collectd listener server that
// accepts datapoints directly from collectd's write_http plugin.  It accepts
// them and sends them out via the common agent datapoint channel.
type Server struct {
	dpChan    chan<- *datapoint.Datapoint
	eventChan chan<- *event.Event
	listener  *collectd.ListenerServer
}

func adaptLogs(keyvals ...interface{}) {
	if keyvals[0] == log.Err {
		logrus.WithError(keyvals[1].(error)).Error("Collectd write server error")
	} else {
		// Just dump it out with spew
		logrus.Info(spew.Sdump(keyvals))
	}
}

func NewServer(ipAddr string, port uint16, dpChan chan<- *datapoint.Datapoint, eventChan chan<- *event.Event) (*Server, error) {
	server := &Server{
		dpChan:    dpChan,
		eventChan: eventChan,
	}

	conf := &collectd.ListenerConfig{
		ListenAddr:      pointer.String(ipAddr + ":" + strconv.Itoa(int(port))),
		ListenPath:      pointer.String("/"),
		Timeout:         pointer.Duration(time.Second * 30),
		HealthCheck:     pointer.String("/healthz"),
		Logger:          log.LoggerFunc(adaptLogs),
		StartingContext: context.Background(),
	}

	var err error
	// Unfortunately this method also starts up the server in a goroutine and
	// only provides error logging if something goes wrong, so it's hard to
	// make this very robust.
	server.listener, err = collectd.NewListener(server, conf)
	if err != nil {
		return nil, err
	}

	return server, nil
}

func (s *Server) Close() error {
	return s.listener.Close()
}

func (s *Server) AddDatapoints(ctx context.Context, dps []*datapoint.Datapoint) error {
	for _, dp := range dps {
		s.dpChan <- dp
	}
	return nil
}

func (s *Server) AddEvents(ctx context.Context, events []*event.Event) error {
	for _, event := range events {
		s.eventChan <- event
	}
	return nil
}
