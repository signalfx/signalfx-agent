package dimensions

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync/atomic"
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
			go rs.processRequests()
		}

		// Block until we can get through a request
		rs.requests <- req
	}
}

func (rs *reqSender) processRequests() {
	atomic.AddInt64(&rs.RunningWorkers, int64(1))
	defer atomic.AddInt64(&rs.RunningWorkers, int64(-1))

	for {
		select {
		case <-rs.ctx.Done():
			return
		case req := <-rs.requests:
			atomic.AddInt64(&rs.TotalRequestsStarted, int64(1))
			if err := rs.sendRequest(req); err != nil {
				atomic.AddInt64(&rs.TotalRequestsFailed, int64(1))
				continue
			}
			atomic.AddInt64(&rs.TotalRequestsCompleted, int64(1))
		}
	}
}

func (rs *reqSender) sendRequest(req *http.Request) error {
	body, statusCode, err := sendRequest(rs.client, req)
	// If it was successful there is nothing else to do.
	if statusCode == 200 {
		onRequestSuccess(req)
		return nil
	}

	if err != nil {
		err = fmt.Errorf("error making HTTP request to %s: %v", req.URL.String(), err)
	} else {
		err = fmt.Errorf("unexpected status code %d on response for request to %s: %s", statusCode, req.URL.String(), string(body))
	}

	onRequestFailed(req, statusCode, err)

	return err
}

type key int

const requestFailedCallbackKey key = 1
const requestSuccessCallbackKey key = 2

type requestFailedCallback func(statusCode int, err error)
type requestSuccessCallback func()

func onRequestSuccess(req *http.Request) {
	ctx := req.Context()
	cb, ok := ctx.Value(requestSuccessCallbackKey).(requestSuccessCallback)
	if !ok {
		return
	}
	cb()
}
func onRequestFailed(req *http.Request, statusCode int, err error) {
	ctx := req.Context()
	cb, ok := ctx.Value(requestFailedCallbackKey).(requestFailedCallback)
	if !ok {
		return
	}
	cb(statusCode, err)
}

func sendRequest(client *http.Client, req *http.Request) ([]byte, int, error) {
	resp, err := client.Do(req)

	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}
