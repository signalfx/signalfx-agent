package dimensions

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/stretchr/testify/require"
)

var putPathRegexp = regexp.MustCompile(`/v2/dimension/([^/]+)/([^/]+)`)
var patchPathRegexp = regexp.MustCompile(`/v2/dimension/([^/]+)/([^/]+)/_/sfxagent`)

type dim struct {
	Key          string            `json:"key"`
	Value        string            `json:"value"`
	Properties   map[string]string `json:"customProperties"`
	Tags         []string          `json:"tags"`
	TagsToRemove []string          `json:"tagsToRemove"`
	WasPatch     bool              `json:"-"`
}

func waitForDims(dimCh <-chan dim, count, waitSeconds int) []dim { // nolint: unparam
	var dims []dim
	timeout := time.After(time.Duration(waitSeconds) * time.Second)

loop:
	for {
		select {
		case dim := <-dimCh:
			dims = append(dims, dim)
			if len(dims) >= count {
				break loop
			}
		case <-timeout:
			break loop
		}
	}

	return dims
}

func makeHandler(dimCh chan<- dim, should500 *atomic.Value) http.HandlerFunc {
	should500.Store(false)

	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if should500.Load().(bool) {
			rw.WriteHeader(500)
			return
		}

		log.Printf("Test server got request: %s", r.URL.Path)
		var re *regexp.Regexp
		switch r.Method {
		case "PUT":
			re = putPathRegexp
		case "PATCH":
			re = patchPathRegexp
		default:
			rw.WriteHeader(404)
			return
		}
		match := re.FindStringSubmatch(r.URL.Path)
		if match == nil {
			rw.WriteHeader(404)
			return
		}

		var bodyDim dim
		if err := json.NewDecoder(r.Body).Decode(&bodyDim); err != nil {
			rw.WriteHeader(400)
			return
		}
		bodyDim.WasPatch = r.Method == "PATCH"

		dimCh <- bodyDim

		rw.WriteHeader(200)
	})
}

func setup() (*DimensionClient, chan dim, *atomic.Value, context.CancelFunc) {
	dimCh := make(chan dim)

	var should500 atomic.Value
	server := httptest.NewServer(makeHandler(dimCh, &should500))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ctx.Done()
		server.Close()
	}()

	client, err := NewDimensionClient(ctx, &config.WriterConfig{
		PropertiesMaxBuffered:      10,
		PropertiesMaxRequests:      10,
		PropertiesSendDelaySeconds: 1,
		PropertiesHistorySize:      1000,
		LogDimensionUpdates:        true,
		APIURL:                     server.URL,
	})
	if err != nil {
		panic("could not make dim client: " + err.Error())
	}
	client.Start()

	return client, dimCh, &should500, cancel
}

func TestDimensionClient(t *testing.T) {
	client, dimCh, should500, cancel := setup()
	defer cancel()

	require.NoError(t, client.AcceptDimension(&types.Dimension{
		Name:  "host",
		Value: "test-box",
		Properties: map[string]string{
			"a": "b",
			"c": "d",
		},
		Tags: map[string]bool{
			"active": true,
		},
		MergeIntoExisting: true,
	}))

	dims := waitForDims(dimCh, 1, 3)
	require.Equal(t, dims, []dim{
		{
			Key:   "host",
			Value: "test-box",
			Properties: map[string]string{
				"a": "b",
				"c": "d",
			},
			Tags:         []string{"active"},
			TagsToRemove: []string{},
			WasPatch:     true,
		},
	})

	// Send the same dimension with different values.
	require.NoError(t, client.AcceptDimension(&types.Dimension{
		Name:  "host",
		Value: "test-box",
		Properties: map[string]string{
			"e": "f",
		},
		Tags: map[string]bool{
			"active": false,
		},
		MergeIntoExisting: true,
	}))

	dims = waitForDims(dimCh, 1, 3)
	require.Equal(t, dims, []dim{
		{
			Key:   "host",
			Value: "test-box",
			Properties: map[string]string{
				"e": "f",
			},
			Tags:         []string{},
			TagsToRemove: []string{"active"},
			WasPatch:     true,
		},
	})

	require.NoError(t, client.AcceptDimension(&types.Dimension{
		Name:  "AWSUniqueID",
		Value: "abcd",
		Properties: map[string]string{
			"a": "b",
		},
		Tags: map[string]bool{
			"is_on": true,
		},
	}))

	dims = waitForDims(dimCh, 1, 3)
	require.Equal(t, dims, []dim{
		{
			Key:   "AWSUniqueID",
			Value: "abcd",
			Properties: map[string]string{
				"a": "b",
			},
			Tags:     []string{"is_on"},
			WasPatch: false,
		},
	})

	should500.Store(true)

	// Send a distinct prop/tag set for same dim with an error
	require.NoError(t, client.AcceptDimension(&types.Dimension{
		Name:  "AWSUniqueID",
		Value: "abcd",
		Properties: map[string]string{
			"a": "b",
			"c": "d",
		},
		Tags: map[string]bool{
			"running": true,
		},
	}))
	dims = waitForDims(dimCh, 1, 3)
	require.Len(t, dims, 0)

	should500.Store(false)
	dims = waitForDims(dimCh, 1, 3)

	// After the server recovers the dim should be resent.
	require.Equal(t, dims, []dim{
		{
			Key:   "AWSUniqueID",
			Value: "abcd",
			Properties: map[string]string{
				"a": "b",
				"c": "d",
			},
			Tags:     []string{"running"},
			WasPatch: false,
		},
	})

	// Send a duplicate
	require.NoError(t, client.AcceptDimension(&types.Dimension{
		Name:  "AWSUniqueID",
		Value: "abcd",
		Properties: map[string]string{
			"a": "b",
			"c": "d",
		},
		Tags: map[string]bool{
			"running": true,
		},
	}))

	dims = waitForDims(dimCh, 1, 3)
	require.Len(t, dims, 0)

	// Send something unique again
	require.NoError(t, client.AcceptDimension(&types.Dimension{
		Name:  "AWSUniqueID",
		Value: "abcd",
		Properties: map[string]string{
			"c": "d",
		},
		Tags: map[string]bool{
			"running": true,
		},
	}))

	dims = waitForDims(dimCh, 1, 3)

	require.Equal(t, dims, []dim{
		{
			Key:   "AWSUniqueID",
			Value: "abcd",
			Properties: map[string]string{
				"c": "d",
			},
			Tags:     []string{"running"},
			WasPatch: false,
		},
	})

	// Send a distinct patch that covers the same prop keys
	require.NoError(t, client.AcceptDimension(&types.Dimension{
		Name:  "host",
		Value: "test-box",
		Properties: map[string]string{
			"a": "z",
		},
		MergeIntoExisting: true,
	}))

	dims = waitForDims(dimCh, 1, 3)
	require.Equal(t, dims, []dim{
		{
			Key:   "host",
			Value: "test-box",
			Properties: map[string]string{
				"a": "z",
			},
			Tags:         []string{},
			TagsToRemove: []string{},
			WasPatch:     true,
		},
	})

	// Send a distinct patch that covers the same tags
	require.NoError(t, client.AcceptDimension(&types.Dimension{
		Name:  "host",
		Value: "test-box",
		Tags: map[string]bool{
			"active": true,
		},
		MergeIntoExisting: true,
	}))

	dims = waitForDims(dimCh, 1, 3)
	require.Equal(t, dims, []dim{
		{
			Key:          "host",
			Value:        "test-box",
			Properties:   map[string]string{},
			Tags:         []string{"active"},
			TagsToRemove: []string{},
			WasPatch:     true,
		},
	})
}

func TestFlappyUpdates(t *testing.T) {
	client, dimCh, _, cancel := setup()
	defer cancel()

	// Do some flappy updates
	for i := 0; i < 5; i++ {
		require.NoError(t, client.AcceptDimension(&types.Dimension{
			Name:  "pod_uid",
			Value: "abcd",
			Properties: map[string]string{
				"index": fmt.Sprintf("%d", i),
			},
		}))
		require.NoError(t, client.AcceptDimension(&types.Dimension{
			Name:  "pod_uid",
			Value: "efgh",
			Properties: map[string]string{
				"index": fmt.Sprintf("%d", i),
			},
			MergeIntoExisting: true,
		}))
	}

	dims := waitForDims(dimCh, 2, 3)
	require.ElementsMatch(t, []dim{
		{
			Key:        "pod_uid",
			Value:      "abcd",
			Properties: map[string]string{"index": "4"},
			WasPatch:   false,
		},
		{
			Key:          "pod_uid",
			Value:        "efgh",
			Properties:   map[string]string{"index": "4"},
			Tags:         []string{},
			TagsToRemove: []string{},
			WasPatch:     true,
		},
	}, dims)

	// Give it enough time to run the counter updates which happen after the
	// request is completed.
	time.Sleep(1 * time.Second)

	require.Equal(t, int64(8), client.TotalFlappyUpdates)
	require.Equal(t, int64(0), client.DimensionsCurrentlyDelayed)
	require.Equal(t, int64(2), client.requestSender.TotalRequestsStarted)
	require.Equal(t, int64(2), client.requestSender.TotalRequestsCompleted)
	require.Equal(t, int64(0), client.requestSender.TotalRequestsFailed)
}
