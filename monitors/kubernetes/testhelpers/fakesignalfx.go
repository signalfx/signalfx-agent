package testhelpers

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/gogo/protobuf/proto"
	sfxproto "github.com/signalfx/com_signalfx_metrics_protobuf"

	"github.com/onsi/gomega"
)

// FakeSignalFx is a mock of the ingest server.  Holds all of the received
// datapoints for later inspection
type FakeSignalFx struct {
	server   *httptest.Server
	received []*sfxproto.DataPoint
	lock     sync.Mutex
}

// NewFakeSignalFx creates a new instance of FakeSignalFx but does not start
// the server
func NewFakeSignalFx() *FakeSignalFx {
	return &FakeSignalFx{
		received: make([]*sfxproto.DataPoint, 0),
	}
}

// Start creates and starts the mock HTTP server
func (f *FakeSignalFx) Start() {
	f.server = httptest.NewUnstartedServer(f)
	f.server.Start()
}

// Close stops the mock HTTP server
func (f *FakeSignalFx) Close() {
	f.server.Close()
}

// URL is the of the mock server to point your objects under test to
func (f *FakeSignalFx) URL() string {
	return f.server.URL
}

// ServeHTTP handles a single request
func (f *FakeSignalFx) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	contents, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	rw.WriteHeader(http.StatusOK)
	io.WriteString(rw, "\"OK\"")

	go func() {
		dpUpload := &sfxproto.DataPointUploadMessage{}
		err := proto.Unmarshal(contents, dpUpload)
		if err != nil {
			panic(fmt.Sprintf("Bad datapoint sent to SignalFx (%s): %#v", err, dpUpload))
		}

		f.lock.Lock()
		f.received = append(f.received, dpUpload.GetDatapoints()...)
		f.lock.Unlock()
	}()
}

// PopIngestedDatapoints returns all currently received datapoints and removes
// them from the server state so that they won't be returned again.
func (f *FakeSignalFx) PopIngestedDatapoints() []*sfxproto.DataPoint {
	gomega.Eventually(func() int { return len(f.received) }, 5).Should(gomega.BeNumerically(">", 0))

	f.lock.Lock()
	defer f.lock.Unlock()

	ret := make([]*sfxproto.DataPoint, 0, len(f.received))
	for _, dp := range f.received {
		ret = append(ret, dp)
	}
	f.received = f.received[:0]
	return ret
}

// EnsureNoDatapoints asserts that there are no datapoints received for 4
// seconds.
func (f *FakeSignalFx) EnsureNoDatapoints() {
	gomega.Consistently(func() int { return len(f.received) }, 4).Should(gomega.Equal(0))
}
