package correlations

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/stretchr/testify/require"
)

var putPathRegexp = regexp.MustCompile(`/v2/apm/correlate/([^/]+)/([^/]+)/([^/]+)`)            // /dimName/dimVal/{service,environment}
var deletePathRegexp = regexp.MustCompile(`/v2/apm/correlate/([^/]+)/([^/]+)/([^/]+)/([^/]+)`) // /dimName/dimValue/{service,environment}/value

func waitForCors(corCh <-chan *Correlation, count, waitSeconds int) []*Correlation { // nolint: unparam
	var cors []*Correlation
	timeout := time.After(time.Duration(waitSeconds) * time.Second)

loop:
	for {
		select {
		case cor := <-corCh:
			cors = append(cors, cor)
			if len(cors) >= count {
				break loop
			}
		case <-timeout:
			break loop
		}
	}

	return cors
}

func makeHandler(corCh chan<- *Correlation, forcedResp *atomic.Value) http.HandlerFunc {
	forcedResp.Store(200)

	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		forcedRespInt := forcedResp.Load().(int)
		if forcedRespInt != 200 {
			rw.WriteHeader(forcedRespInt)
			return
		}

		log.Printf("Test server got %s request: %s", r.Method, r.URL.Path)
		var cor *Correlation
		switch r.Method {
		case "PUT":
			match := putPathRegexp.FindStringSubmatch(r.URL.Path)
			if match == nil || len(match) < 4 {
				rw.WriteHeader(404)
				return
			}

			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				rw.WriteHeader(400)
				return
			}
			cor = &Correlation{
				Operation: Put,
				DimName:   match[1],
				DimValue:  match[2],
				Type:      Type(match[3]),
				Value:     string(body),
			}
		case "DELETE":
			match := deletePathRegexp.FindStringSubmatch(r.URL.Path)
			if match == nil || len(match) < 5 {
				rw.WriteHeader(404)
				return
			}
			cor = &Correlation{
				Operation: Delete,
				DimName:   match[1],
				DimValue:  match[2],
				Type:      Type(match[3]),
				Value:     match[4],
			}
		default:
			rw.WriteHeader(404)
			return
		}

		corCh <- cor

		rw.WriteHeader(200)
	})
}

func setup() (CorrelationClient, chan *Correlation, *atomic.Value, context.CancelFunc) {
	serverCh := make(chan *Correlation, 100)

	var forcedResp atomic.Value
	server := httptest.NewServer(makeHandler(serverCh, &forcedResp))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ctx.Done()
		server.Close()
	}()

	client, err := NewCorrelationClient(ctx, &config.WriterConfig{
		PropertiesMaxBuffered:      10,
		PropertiesMaxRequests:      10,
		PropertiesSendDelaySeconds: 1,
		PropertiesHistorySize:      1000,
		LogDimensionUpdates:        true,
		APIURL:                     server.URL,
	})
	if err != nil {
		panic("could not make correlation client: " + err.Error())
	}
	client.Start()

	return client, serverCh, &forcedResp, cancel
}

func TestCorrelationClient(t *testing.T) {
	client, serverCh, forcedResp, cancel := setup()
	defer close(serverCh)
	defer cancel()

	for _, correlationType := range []Type{Service, Environment} {
		for _, op := range []Operation{Put, Delete} {
			correlationType := correlationType
			op := op
			t.Run(fmt.Sprintf("%v %v", op, correlationType), func(t *testing.T) {
				testData := &Correlation{Type: correlationType, Operation: op, DimName: "host", DimValue: "test-box", Value: "test-service"}
				client.AcceptCorrelation(testData)
				cors := waitForCors(serverCh, 1, 5)
				require.Equal(t, []*Correlation{testData}, cors)
			})
		}
	}
	t.Run("does not retry 4xx responses", func(t *testing.T) {
		forcedResp.Store(400)

		testData := &Correlation{Type: Service, Operation: Put, DimName: "host", DimValue: "test-box", Value: "test-service"}
		client.AcceptCorrelation(testData)

		cors := waitForCors(serverCh, 1, 3)
		require.Len(t, cors, 0)

		forcedResp.Store(200)
		cors = waitForCors(serverCh, 1, 3)
		require.Len(t, cors, 0)
	})
	t.Run("does retry 500 responses", func(t *testing.T) {
		forcedResp.Store(500)

		testData := &Correlation{Type: Service, Operation: Put, DimName: "host", DimValue: "test-box", Value: "test-service"}
		client.AcceptCorrelation(testData)

		cors := waitForCors(serverCh, 1, 2)
		require.Len(t, cors, 0)

		forcedResp.Store(200)
		cors = waitForCors(serverCh, 1, 2)
		require.Equal(t, cors, []*Correlation{testData})
	})
}
