package testhelpers

import (
	"fmt"
    "io"
    "io/ioutil"
    "net/http"
    "net/http/httptest"
	"sync"

    sfxproto "github.com/signalfx/com_signalfx_metrics_protobuf"
    "github.com/gogo/protobuf/proto"

    . "github.com/onsi/gomega"
)

type FakeSignalFx struct {
    server           *httptest.Server
    received         []*sfxproto.DataPoint
	lock             sync.Mutex
}

func NewFakeSignalFx() *FakeSignalFx {
    return &FakeSignalFx{
        received: make([]*sfxproto.DataPoint, 0),
    }
}

func (f *FakeSignalFx) Start() {
    f.server = httptest.NewUnstartedServer(f)
    f.server.Start()
}

func (f *FakeSignalFx) Close() {
    f.server.Close()
}

func (f *FakeSignalFx) URL() string {
    return f.server.URL
}

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

func (f *FakeSignalFx) PopIngestedDatapoints() []*sfxproto.DataPoint {
    Eventually(func() int { return len(f.received) }, 5).Should(BeNumerically(">", 0))

	f.lock.Lock()
	defer f.lock.Unlock()

	ret := make([]*sfxproto.DataPoint, 0, len(f.received))
	for _, dp := range f.received {
		ret = append(ret, dp)
	}
	f.received = f.received[:0]
    return ret
}

func (f *FakeSignalFx) EnsureNoDatapoints() {
    Consistently(func() int { return len(f.received) }, 4).Should(Equal(0))
}
