package protocol

import (
	"github.com/gorilla/mux"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/log"
	"github.com/signalfx/golib/sfxclient"
	"net/http"
	"sync/atomic"
)

// CloseableHealthCheck is a helper class intended to be used as an anonymous field
type CloseableHealthCheck struct {
	setCloseHeader    int32
	totalHealthChecks int64
}

// CloseHealthCheck is called to change the status of the healthcheck from 200 to 404 and close the connection
func (c *CloseableHealthCheck) CloseHealthCheck() {
	atomic.AddInt32(&c.setCloseHeader, 1)
}

// HealthDatapoints returns the total health checks done
func (c *CloseableHealthCheck) HealthDatapoints() []*datapoint.Datapoint {
	return []*datapoint.Datapoint{sfxclient.Cumulative("total_health_checks", nil, atomic.LoadInt64(&c.totalHealthChecks))}
}

// SetupHealthCheck sets up a closeable healthcheck, when open returns 200, when closed returns 404 and close the connection
func (c *CloseableHealthCheck) SetupHealthCheck(healthCheck *string, r *mux.Router, logger log.Logger) {
	f := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&c.totalHealthChecks, 1)
		if atomic.LoadInt32(&c.setCloseHeader) != 0 {
			rw.Header().Set("Connection", "Close")
			rw.WriteHeader(http.StatusNotFound)
			_, err := rw.Write([]byte("graceful shutdown"))
			log.IfErr(logger, err)
			return
		}
		_, err := rw.Write([]byte("OK"))
		log.IfErr(logger, err)
	})
	r.Handle(*healthCheck, f)
}
