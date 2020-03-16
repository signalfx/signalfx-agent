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

var getPathRegexp = regexp.MustCompile(`/v2/apm/correlate/([^/]+)/([^/]+)`)                    // /dimName/dimVal
var putPathRegexp = regexp.MustCompile(`/v2/apm/correlate/([^/]+)/([^/]+)/([^/]+)`)            // /dimName/dimVal/{service,environment}
var deletePathRegexp = regexp.MustCompile(`/v2/apm/correlate/([^/]+)/([^/]+)/([^/]+)/([^/]+)`) // /dimName/dimValue/{service,environment}/value

func waitForCors(corCh <-chan *request, count, waitSeconds int) []*request { // nolint: unparam
	var cors []*request
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

func makeHandler(corCh chan<- *request, forcedResp *atomic.Value) http.HandlerFunc {
	forcedResp.Store(200)

	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		forcedRespInt := forcedResp.Load().(int)
		if forcedRespInt != 200 {
			rw.WriteHeader(forcedRespInt)
			return
		}

		log.Printf("Test server got %s request: %s", r.Method, r.URL.Path)
		var cor *request
		switch r.Method {
		case http.MethodGet:
			match := getPathRegexp.FindStringSubmatch(r.URL.Path)
			if match == nil || len(match) < 3 {
				rw.WriteHeader(404)
				return
			}
			cor = &request{
				operation: r.Method,
				Correlation: &Correlation{
					DimName:  match[1],
					DimValue: match[2],
				},
			}
		case http.MethodPut:
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
			cor = &request{
				operation: r.Method,
				Correlation: &Correlation{
					DimName:  match[1],
					DimValue: match[2],
					Type:     Type(match[3]),
					Value:    string(body),
				},
			}

		case http.MethodDelete:
			match := deletePathRegexp.FindStringSubmatch(r.URL.Path)
			if match == nil || len(match) < 5 {
				rw.WriteHeader(404)
				return
			}
			cor = &request{
				operation: r.Method,
				Correlation: &Correlation{
					DimName:  match[1],
					DimValue: match[2],
					Type:     Type(match[3]),
					Value:    match[4],
				},
			}
		default:
			rw.WriteHeader(404)
			return
		}

		corCh <- cor

		rw.WriteHeader(200)
	})
}

func setup() (CorrelationClient, chan *request, *atomic.Value, context.CancelFunc) {
	serverCh := make(chan *request, 100)

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
		for _, op := range []string{http.MethodPut, http.MethodDelete} {
			op := op
			correlationType := correlationType
			t.Run(fmt.Sprintf("%v %v", op, correlationType), func(t *testing.T) {
				testData := &Correlation{Type: correlationType, DimName: "host", DimValue: "test-box", Value: "test-service"}
				switch op {
				case http.MethodPut:
					client.Correlate(testData)
				case http.MethodDelete:
					client.Delete(testData)
				}
				cors := waitForCors(serverCh, 1, 5)
				require.Equal(t, []*request{&request{operation: op, Correlation: testData}}, cors)
			})
		}
	}
	t.Run("does not retry 4xx responses", func(t *testing.T) {
		forcedResp.Store(400)

		testData := &Correlation{Type: Service, DimName: "host", DimValue: "test-box", Value: "test-service"}
		client.Correlate(testData)

		cors := waitForCors(serverCh, 1, 3)
		require.Len(t, cors, 0)

		forcedResp.Store(200)
		cors = waitForCors(serverCh, 1, 3)
		require.Len(t, cors, 0)
	})
	t.Run("does retry 500 responses", func(t *testing.T) {
		forcedResp.Store(500)

		testData := &Correlation{Type: Service, DimName: "host", DimValue: "test-box", Value: "test-service"}
		client.Correlate(testData)

		cors := waitForCors(serverCh, 1, 2)
		require.Len(t, cors, 0)

		forcedResp.Store(200)
		cors = waitForCors(serverCh, 1, 2)
		require.Equal(t, []*request{&request{Correlation: testData, operation: http.MethodPut}}, cors)
	})
}
