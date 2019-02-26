package properties

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync/atomic"

	log "github.com/sirupsen/logrus"
)

type reqSender struct {
	client      *http.Client
	requests    chan *http.Request
	workerCount uint
	ctx         context.Context

	RunningWorkers         int64
	TotalRequestsStarted   int64
	TotalRequestsCompleted int64
	TotalRequestsFailed    int64
}

func newReqSender(ctx context.Context, client *http.Client, workerCount uint) *reqSender {
	return &reqSender{
		client: client,
		// Unbuffered so that it blocks clients
		requests:    make(chan *http.Request),
		workerCount: workerCount,
		ctx:         ctx,
	}
}

// Not thread-safe
func (rs *reqSender) send(req *http.Request) {
	// Slight optimization to avoid spinning up unnecessary workers if there
	// aren't ever that many dim updates. Once workers start, they remain for the
	// duration of the agent.
	select {
	case rs.requests <- req:
		return
	default:
		if atomic.LoadInt64(&rs.RunningWorkers) < int64(rs.workerCount) {
			go rs.processRequests(rs.ctx)
		}

		// Block until we can get through a request
		rs.requests <- req
	}
	return
}

type writerKey int

var reqDoneCallbackKeyVar writerKey

func (rs *reqSender) processRequests(ctx context.Context) error {
	atomic.AddInt64(&rs.RunningWorkers, int64(1))
	defer atomic.AddInt64(&rs.RunningWorkers, int64(-1))

	for {
		select {
		case <-ctx.Done():
			return nil
		case req := <-rs.requests:
			atomic.AddInt64(&rs.TotalRequestsStarted, int64(1))
			if err := sendRequest(rs.client, req); err != nil {
				atomic.AddInt64(&rs.TotalRequestsFailed, int64(1))
				log.WithError(err).WithField("url", req.URL.String()).Error("Unable to update dimension")
				continue
			}
			atomic.AddInt64(&rs.TotalRequestsCompleted, int64(1))

			if cb := req.Context().Value(reqDoneCallbackKeyVar); cb != nil {
				cb.(func())()
			}
		}
	}
}

func sendRequest(client *http.Client, req *http.Request) error {
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d on response %s", resp.StatusCode, string(body))
	}

	return nil
}
