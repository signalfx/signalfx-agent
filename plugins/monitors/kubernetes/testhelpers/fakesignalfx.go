package testhelpers

import (
    "io"
    "io/ioutil"
    "net/http"
    "net/http/httptest"

    sfxproto "github.com/signalfx/com_signalfx_metrics_protobuf"
    "github.com/gogo/protobuf/proto"

    . "github.com/onsi/gomega"
)

type FakeSignalFx struct {
    server           *httptest.Server
    ReceivedContents chan []byte
}

func NewFakeSignalFx() *FakeSignalFx {
    return &FakeSignalFx{
        ReceivedContents: make(chan []byte, 100),
    }
}

func (f *FakeSignalFx) Start() {
    f.server = httptest.NewUnstartedServer(f)
    f.server.Start()
}

func (f *FakeSignalFx) Close() {
    f.server.Close()
	for len(f.ReceivedContents) > 0 {
		<-f.ReceivedContents
	}
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
        f.ReceivedContents <- contents
    }()
}

func (f *FakeSignalFx) GetIngestedDatapoints() []*sfxproto.DataPoint {
    var contents []byte
    Eventually(f.ReceivedContents, 5).Should(Receive(&contents))

    dpUpload := &sfxproto.DataPointUploadMessage{}
    err := proto.Unmarshal(contents, dpUpload)
    Expect(err).ToNot(HaveOccurred())

    return dpUpload.GetDatapoints()
}

func (f *FakeSignalFx) EnsureNoDatapoints() {
    Consistently(f.ReceivedContents, 4).ShouldNot(Receive())
}
