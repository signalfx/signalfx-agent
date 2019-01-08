package signalfx

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/gorilla/mux"
	jThrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/signalfx/golib/log"
	"github.com/signalfx/golib/pointer"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/golib/trace"
	"github.com/signalfx/golib/web"
)

// JaegerThriftTraceDecoderV1 decodes Jaeger thrift spans to structs
type JaegerThriftTraceDecoderV1 struct {
	Logger          log.Logger
	Sink            trace.Sink
	protocolFactory *thrift.TBinaryProtocolFactory
	bufferPool      sync.Pool
}

// NewJaegerThriftTraceDecoderV1 creates a new decoder for Jaeger Thrift spans
func NewJaegerThriftTraceDecoderV1(logger log.Logger, sink trace.Sink) *JaegerThriftTraceDecoderV1 {
	return &JaegerThriftTraceDecoderV1{
		Logger:          logger,
		Sink:            sink,
		protocolFactory: thrift.NewTBinaryProtocolFactoryDefault(),
		bufferPool: sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, 2048))
			},
		},
	}

}

var errInvalidJaegerTraceFormat = errors.New("invalid Jaeger format")

const (
	// JaegerV1 binary thrift protocol
	JaegerV1 = "jaeger_thrift_v1"
)

func setupThriftTraceV1(ctx context.Context, r *mux.Router, sink Sink, logger log.Logger, httpChain web.NextConstructor) sfxclient.Collector {
	handler, st := SetupChain(ctx, sink, JaegerV1, func(s Sink) ErrorReader {
		return NewJaegerThriftTraceDecoderV1(logger, sink)
	}, httpChain, logger)

	SetupThriftByPaths(r, handler, DefaultTracePathV1)
	return st
}

// SetupThriftByPaths tells the router which paths the given handler (which should handle the given endpoint) should see
func SetupThriftByPaths(r *mux.Router, handler http.Handler, endpoint string) {
	r.Path(endpoint).Methods("POST").Headers("Content-Type", "application/x-thrift").Handler(handler)
	r.Path(endpoint).Methods("POST").Headers("Content-Type", "application/vnd.apache.thrift.binary").Handler(handler)
}

// Code inspired by
// https://github.com/jaegertracing/jaeger/blob/89f3ccaef21d256728f02ec9d73b31f9c3bde71a/cmd/collector/app/http_handler.go#L61
func (decoder *JaegerThriftTraceDecoderV1) Read(ctx context.Context, req *http.Request) error {
	buf := decoder.bufferPool.Get().(*bytes.Buffer)
	defer decoder.bufferPool.Put(buf)
	buf.Reset()

	_, err := io.Copy(buf, req.Body)
	if err != nil {
		return errors.New("could not read request body")
	}

	protocol := decoder.protocolFactory.GetProtocol(&thrift.TMemoryBuffer{
		Buffer: buf,
	})
	batch := jThrift.Batch{}
	if err := batch.Read(protocol); err != nil {
		return errInvalidJaegerTraceFormat
	}

	spans := convertJaegerBatch(&batch)

	return decoder.Sink.AddSpans(ctx, spans)
}

func convertJaegerBatch(batch *jThrift.Batch) []*trace.Span {
	spans := make([]*trace.Span, len(batch.Spans))
	for i := range batch.Spans {
		spans[i] = convertJaegerSpan(batch.Spans[i], batch.Process)
	}

	return spans
}

func convertJaegerSpan(tSpan *jThrift.Span, tProcess *jThrift.Process) *trace.Span {
	var ptrParentID *string
	if tSpan.ParentSpanId != 0 {
		ptrParentID = pointer.String(padID(strconv.FormatUint(uint64(tSpan.ParentSpanId), 16)))
	}

	localEndpoint := &trace.Endpoint{
		ServiceName: &tProcess.ServiceName,
	}

	var ptrDebug *bool
	if tSpan.Flags&2 > 0 {
		ptrDebug = pointer.Bool(true)
	}

	kind, remoteEndpoint, tags := processJaegerTags(tSpan)

	for _, t := range tProcess.Tags {
		if t.Key == "ip" && t.VStr != nil {
			localEndpoint.Ipv4 = t.VStr
		} else {
			tags[t.Key] = tagValueToString(t)
		}
	}

	traceID := padID(strconv.FormatUint(uint64(tSpan.TraceIdLow), 16))
	if tSpan.TraceIdHigh != 0 {
		traceID = padID(strconv.FormatUint(uint64(tSpan.TraceIdHigh), 16) + traceID)
	}

	span := &trace.Span{
		TraceID:        traceID,
		ID:             padID(strconv.FormatUint(uint64(tSpan.SpanId), 16)),
		ParentID:       ptrParentID,
		Debug:          ptrDebug,
		Name:           &tSpan.OperationName,
		Timestamp:      &tSpan.StartTime,
		Duration:       &tSpan.Duration,
		Kind:           kind,
		LocalEndpoint:  localEndpoint,
		RemoteEndpoint: remoteEndpoint,
		Annotations:    convertJaegerLogs(tSpan.Logs),
		Tags:           tags,
	}
	return span
}

func convertJaegerLogs(logs []*jThrift.Log) []*trace.Annotation {
	annotations := make([]*trace.Annotation, 0, len(logs))
	for i := range logs {
		anno := trace.Annotation{
			Timestamp: &logs[i].Timestamp,
		}
		if content, err := materializeWithJSON(logs[i].Fields); err == nil {
			anno.Value = pointer.String(string(content))
		}
		annotations = append(annotations, &anno)
	}
	return annotations
}

// Handle special tags that get converted to the kind and remote endpoint
// fields, and throw the rest of the tags into a map that becomes the Zipkin
// Tags field.
// nolint: gocyclo
func processJaegerTags(s *jThrift.Span) (*string, *trace.Endpoint, map[string]string) {
	var kind *string
	var remote *trace.Endpoint
	tags := make(map[string]string, len(s.Tags))

	ensureRemote := func() {
		if remote == nil {
			remote = &trace.Endpoint{}
		}
	}

	for i := range s.Tags {
		switch s.Tags[i].Key {
		case string(ext.PeerHostIPv4):
			ip := convertPeerIPv4(s.Tags[i])
			if ip == "" {
				continue
			}
			ensureRemote()
			remote.Ipv4 = pointer.String(ip)
		// ipv6 host is always string
		case string(ext.PeerHostIPv6):
			if s.Tags[i].VStr != nil {
				ensureRemote()
				remote.Ipv6 = s.Tags[i].VStr
			}
		case string(ext.PeerPort):
			port := convertPeerPort(s.Tags[i])
			if port == 0 {
				continue
			}
			ensureRemote()
			remote.Port = &port
		case string(ext.PeerService):
			ensureRemote()
			remote.ServiceName = s.Tags[i].VStr
		case string(ext.SpanKind):
			kind = convertKind(s.Tags[i])
		default:
			val := tagValueToString(s.Tags[i])
			if val != "" {
				tags[s.Tags[i].Key] = val
			}
		}
	}
	return kind, remote, tags
}

func convertKind(tag *jThrift.Tag) *string {
	var kind *string
	switch tag.GetVStr() {
	case string(ext.SpanKindRPCClientEnum):
		kind = &ClientKind
	case string(ext.SpanKindRPCServerEnum):
		kind = &ServerKind
	case string(ext.SpanKindProducerEnum):
		kind = &ProducerKind
	case string(ext.SpanKindConsumerEnum):
		kind = &ConsumerKind
	}
	return kind
}

func convertPeerIPv4(tag *jThrift.Tag) string {
	switch tag.VType {
	case jThrift.TagType_STRING:
		if ip := net.ParseIP(tag.GetVStr()); ip != nil {
			return ip.To4().String()
		}
	case jThrift.TagType_LONG:
		localIP := make(net.IP, 4)
		binary.BigEndian.PutUint32(localIP, uint32(tag.GetVLong()))
		return localIP.String()
	}
	return ""
}

func convertPeerPort(tag *jThrift.Tag) int32 {
	switch tag.VType {
	case jThrift.TagType_STRING:
		if port, err := strconv.ParseUint(tag.GetVStr(), 10, 16); err == nil {
			return int32(port)
		}
	case jThrift.TagType_LONG:
		return int32(tag.GetVLong())
	}
	return 0
}

// materializeWithJSON converts log Fields into JSON string, or just the field
// value of the event field, if present.
func materializeWithJSON(logFields []*jThrift.Tag) ([]byte, error) {
	fields := make(map[string]string, len(logFields))
	for i := range logFields {
		fields[logFields[i].Key] = tagValueToString(logFields[i])
	}
	if event, ok := fields["event"]; ok && len(fields) == 1 {
		return []byte(event), nil
	}
	return json.Marshal(fields)
}

var pads = []string{
	"",
	"0",
	"00",
	"000",
	"0000",
	"00000",
	"000000",
	"0000000",
	"00000000",
	"000000000",
	"0000000000",
	"00000000000",
	"000000000000",
	"0000000000000",
	"00000000000000",
	"000000000000000",
}

// The way IDs get converted to strings in some of the jaeger code, leading 0s
// can be dropped, which will cause the ids to fail validation on our backend.
func padID(id string) string {
	expectedLen := 0
	if len(id) < 16 {
		expectedLen = 16
	} else if len(id) > 16 && len(id) < 32 {
		expectedLen = 32
	} else {
		return id
	}

	return pads[expectedLen-len(id)] + id
}

func tagValueToString(tag *jThrift.Tag) string {
	switch tag.VType {
	case jThrift.TagType_STRING:
		return tag.GetVStr()
	case jThrift.TagType_DOUBLE:
		return strconv.FormatFloat(tag.GetVDouble(), 'f', -1, 64)
	case jThrift.TagType_BOOL:
		if tag.GetVBool() {
			return "true"
		}
		return "false"
	case jThrift.TagType_LONG:
		return strconv.FormatInt(tag.GetVLong(), 10)
	default:
		return ""
	}
}
