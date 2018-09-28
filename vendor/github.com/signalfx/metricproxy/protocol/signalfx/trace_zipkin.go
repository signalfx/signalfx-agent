package signalfx

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mailru/easyjson"
	"github.com/signalfx/golib/errors"
	"github.com/signalfx/golib/log"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/golib/trace"
	"github.com/signalfx/golib/web"
	"github.com/signalfx/metricproxy/protocol/signalfx/format"
)

const (
	// DefaultTracePathV1 is the default listen path
	DefaultTracePathV1 = "/v1/trace"
	// ZipkinV1 is a constant used for protocol naming
	ZipkinV1 = "zipkin_json_v1"
)

// Constants as variables so it is easy to get a pointer to them
var (
	trueVar = true

	ClientKind   = "CLIENT"
	ServerKind   = "SERVER"
	ProducerKind = "PRODUCER"
	ConsumerKind = "CONSUMER"
)

// InputSpan is an alias
type InputSpan signalfxformat.InputSpan

func (is *InputSpan) isDefinitelyZipkinV2() bool {
	// The presence of the "kind" field, tags, or local/remote endpoints is a
	// dead giveaway that this is a Zipkin v2 span, so shortcut the whole
	// process and return it as an optimization.  If it doesn't have any of
	// those things it could still be a V2 span since none of them are strictly
	// required to be there.
	return is.Span.Kind != nil || len(is.Span.Tags) > 0 || is.Span.LocalEndpoint != nil || is.Span.RemoteEndpoint != nil
}

// asZipkinV2 shortcuts the span conversion process and treats the InputSpan as
// ZipkinV2 and returns that span directly.
func (is *InputSpan) fromZipkinV2() (*trace.Span, error) {
	// Do some basic validation
	if len(is.BinaryAnnotations) > 0 {
		return nil, errors.New("span cannot have binaryAnnotations with Zipkin V2 fields")
	}

	if len(is.Annotations) > 0 {
		is.Span.Annotations = make([]*trace.Annotation, len(is.Annotations))
		for i := range is.Annotations {
			is.Span.Annotations[i] = is.Annotations[i].ToV2()
		}
	}
	is.Span.ParentID = normalizeParentSpanID(is.Span.ParentID)

	return &is.Span, nil
}

// asTraceSpan should be used when we are not sure that the InputSpan is
// already in Zipkin V2 format.  It returns a slice of our SignalFx span
// object, which is equivalent to a Zipkin V2 span.  A single span in Zipkin v1
// can contain multiple v2 spans because the annotations and binary annotations
// contain endpoints.  This would also work for Zipkin V2 spans, it just
// involves a lot more processing.  The conversion code was mostly ported from
// https://github.com/openzipkin/zipkin/blob/2.8.4/zipkin/src/main/java/zipkin/internal/V2SpanConverter.java
func (is *InputSpan) fromZipkinV1() ([]*trace.Span, error) {
	if is.Span.Tags == nil {
		is.Span.Tags = map[string]string{}
	}

	is.Span.ParentID = normalizeParentSpanID(is.Span.ParentID)

	spanCopy := is.Span
	spanBuilder := &spanBuilder{
		spans: []*trace.Span{&spanCopy},
	}
	spanBuilder.processAnnotations(is)
	if err := spanBuilder.processBinaryAnnotations(is); err != nil {
		return nil, err
	}

	return spanBuilder.spans, nil
}

func (is *InputSpan) endTimestampReflectsSpanDuration(end *signalfxformat.InputAnnotation) bool {
	return end != nil && is.Timestamp != nil && is.Duration != nil && end.Timestamp != nil &&
		*is.Timestamp+*is.Duration == *end.Timestamp
}

type spanBuilder struct {
	spans                          []*trace.Span
	cs, sr, ss, cr, ms, mr, ws, wr *signalfxformat.InputAnnotation
}

func (sb *spanBuilder) addSpanForEndpoint(is *InputSpan, e *trace.Endpoint) *trace.Span {
	s := is.Span
	s.LocalEndpoint = e
	s.Tags = map[string]string{}

	sb.spans = append(sb.spans, &s)
	return &s
}

func (sb *spanBuilder) spanForEndpoint(is *InputSpan, e *trace.Endpoint) *trace.Span {
	if e == nil {
		// Allocate missing endpoint data to first span.  For a Zipkin v2
		// span this will be the only one.
		return sb.spans[0]
	}

	for i := range sb.spans {
		next := sb.spans[i]
		if next.LocalEndpoint == nil {
			next.LocalEndpoint = e
			return next
		} else if closeEnough(next.LocalEndpoint, e) {
			return next
		}
	}

	return sb.addSpanForEndpoint(is, e)
}

func (sb *spanBuilder) processAnnotations(is *InputSpan) {
	sb.pullOutSpecialAnnotations(is)
	sb.fillInStartAnnotations(is)

	if sb.cs != nil && sb.sr != nil {
		sb.fillInMissingTimings(is)
	} else if sb.cs != nil && sb.cr != nil {
		sb.maybeTimestampDuration(sb.cs, sb.cr, is)
	} else if sb.sr != nil && sb.ss != nil {
		sb.maybeTimestampDuration(sb.sr, sb.ss, is)
	} else { // otherwise, the span is incomplete. revert special-casing
		sb.handleIncompleteSpan(is)
	}

	// Span v1 format did not have a shared flag. By convention, span.timestamp being absent
	// implied shared. When we only see the server-side, carry this signal over.
	if sb.cs == nil && (sb.sr != nil && is.Timestamp == nil) {
		sb.spanForEndpoint(is, sb.sr.Endpoint).Shared = &trueVar
	}

	sb.handleMessageQueueAnnotations(is)
}

func (sb *spanBuilder) pullOutSpecialAnnotations(is *InputSpan) {
	for i := range is.Annotations {
		anno := is.Annotations[i]

		span := sb.spanForEndpoint(is, anno.Endpoint)

		var processed bool
		// core annotations require an endpoint. Don't give special treatment when that's missing
		if anno.Value != nil && len(*anno.Value) == 2 && anno.Endpoint != nil {
			processed = sb.handleSpecialAnnotation(anno, span)
		} else {
			processed = false
		}

		if !processed {
			span.Annotations = append(span.Annotations, &trace.Annotation{
				Timestamp: anno.Timestamp,
				Value:     anno.Value,
			})
		}
	}
}

func (sb *spanBuilder) handleSpecialAnnotation(anno *signalfxformat.InputAnnotation, span *trace.Span) bool {
	switch *anno.Value {
	case "cs":
		span.Kind = &ClientKind
		sb.cs = anno
	case "sr":
		span.Kind = &ServerKind
		sb.sr = anno
	case "ss":
		span.Kind = &ServerKind
		sb.ss = anno
	case "cr":
		span.Kind = &ClientKind
		sb.cr = anno
	case "ms":
		span.Kind = &ProducerKind
		sb.ms = anno
	case "mr":
		span.Kind = &ConsumerKind
		sb.mr = anno
	case "ws":
		sb.ws = anno
	case "wr":
		sb.wr = anno
	default:
		return false
	}
	return true
}

func (sb *spanBuilder) fillInStartAnnotations(is *InputSpan) {
	// When bridging between event and span model, you can end up missing a start annotation
	if sb.cs == nil && is.endTimestampReflectsSpanDuration(sb.cr) {
		val := "cs"
		sb.cs = &signalfxformat.InputAnnotation{
			Timestamp: is.Timestamp,
			Value:     &val,
			Endpoint:  sb.cr.Endpoint,
		}
	}
	if sb.sr == nil && is.endTimestampReflectsSpanDuration(sb.ss) {
		val := "sr"
		sb.sr = &signalfxformat.InputAnnotation{
			Timestamp: is.Timestamp,
			Value:     &val,
			Endpoint:  sb.ss.Endpoint,
		}
	}
}

func (sb *spanBuilder) fillInMissingTimings(is *InputSpan) {
	// in a shared span, the client side owns span duration by annotations or explicit timestamp
	sb.maybeTimestampDuration(sb.cs, sb.cr, is)

	// special-case loopback: We need to make sure on loopback there are two span2s
	client := sb.spanForEndpoint(is, sb.cs.Endpoint)

	var server *trace.Span
	if closeEnough(sb.cs.Endpoint, sb.sr.Endpoint) {
		client.Kind = &ClientKind
		// fork a new span for the server side
		server = sb.addSpanForEndpoint(is, sb.sr.Endpoint)
		server.Kind = &ServerKind
	} else {
		server = sb.spanForEndpoint(is, sb.sr.Endpoint)
	}

	// the server side is smaller than that, we have to read annotations to find out
	server.Shared = &trueVar
	server.Timestamp = sb.sr.Timestamp
	if sb.ss != nil && sb.ss.Timestamp != nil && sb.sr.Timestamp != nil {
		ts := *sb.ss.Timestamp - *sb.sr.Timestamp
		server.Duration = &ts
	}
	if sb.cr == nil && is.Duration == nil {
		client.Duration = nil
	}
}

func (sb *spanBuilder) handleIncompleteSpan(is *InputSpan) {
	for i := range sb.spans {
		next := sb.spans[i]
		if next.Kind != nil && *next.Kind == ClientKind {
			if sb.cs != nil {
				next.Timestamp = sb.cs.Timestamp
			}
			if sb.cr != nil {
				next.Annotations = append(next.Annotations, &trace.Annotation{
					Timestamp: sb.cr.Timestamp,
					Value:     sb.cr.Value,
				})
			}
		} else if next.Kind != nil && *next.Kind == ServerKind {
			if sb.sr != nil {
				next.Timestamp = sb.sr.Timestamp
			}
			if sb.ss != nil {
				next.Annotations = append(next.Annotations, &trace.Annotation{
					Timestamp: sb.ss.Timestamp,
					Value:     sb.ss.Value,
				})
			}
		}

		sb.fillInTimingsOnFirstSpan(is)
	}
}

func (sb *spanBuilder) fillInTimingsOnFirstSpan(is *InputSpan) {
	if is.Timestamp != nil {
		sb.spans[0].Timestamp = is.Timestamp
		sb.spans[0].Duration = is.Duration
	}
}

func (sb *spanBuilder) handleMessageQueueAnnotations(is *InputSpan) {
	// ms and mr are not supposed to be in the same span, but in case they are..
	if sb.ms != nil && sb.mr != nil {
		sb.handleBothMSAndMR(is)
	} else if sb.ms != nil {
		sb.maybeTimestampDuration(sb.ms, sb.ws, is)
	} else if sb.mr != nil {
		if sb.wr != nil {
			sb.maybeTimestampDuration(sb.wr, sb.mr, is)
		} else {
			sb.maybeTimestampDuration(sb.mr, nil, is)
		}
	} else {
		if sb.ws != nil {
			span := sb.spanForEndpoint(is, sb.ws.Endpoint)
			span.Annotations = append(span.Annotations, &trace.Annotation{
				Timestamp: sb.ws.Timestamp,
				Value:     sb.ws.Value,
			})
		}
		if sb.wr != nil {
			span := sb.spanForEndpoint(is, sb.wr.Endpoint)
			span.Annotations = append(span.Annotations, &trace.Annotation{
				Timestamp: sb.wr.Timestamp,
				Value:     sb.wr.Value,
			})
		}
	}
}

func (sb *spanBuilder) handleBothMSAndMR(is *InputSpan) {
	// special-case loopback: We need to make sure on loopback there are two span2s
	producer := sb.spanForEndpoint(is, sb.ms.Endpoint)
	var consumer *trace.Span
	if closeEnough(sb.ms.Endpoint, sb.mr.Endpoint) {
		producer.Kind = &ProducerKind
		// fork a new span for the consumer side
		consumer = sb.addSpanForEndpoint(is, sb.mr.Endpoint)
		consumer.Kind = &ConsumerKind
	} else {
		consumer = sb.spanForEndpoint(is, sb.mr.Endpoint)
	}

	consumer.Shared = &trueVar
	if sb.wr != nil && sb.mr.Timestamp != nil && sb.wr.Timestamp != nil {
		consumer.Timestamp = sb.wr.Timestamp
		ts := *sb.mr.Timestamp - *sb.wr.Timestamp
		consumer.Duration = &ts
	} else {
		consumer.Timestamp = sb.mr.Timestamp
	}

	producer.Timestamp = sb.ms.Timestamp
	if sb.ws != nil && sb.ws.Timestamp != nil && sb.ms.Timestamp != nil {
		ts := *sb.ws.Timestamp - *sb.ms.Timestamp
		producer.Duration = &ts
	}
}

func (sb *spanBuilder) maybeTimestampDuration(begin, end *signalfxformat.InputAnnotation, is *InputSpan) {
	span2 := sb.spanForEndpoint(is, begin.Endpoint)

	if is.Timestamp != nil && is.Duration != nil {
		span2.Timestamp = is.Timestamp
		span2.Duration = is.Duration
	} else {
		span2.Timestamp = begin.Timestamp
		if end != nil && end.Timestamp != nil && begin.Timestamp != nil {
			ts := *end.Timestamp - *begin.Timestamp
			span2.Duration = &ts
		}
	}
}

func (sb *spanBuilder) processBinaryAnnotations(is *InputSpan) error {
	ca, sa, ma, err := sb.pullOutSpecialBinaryAnnotations(is)
	if err != nil {
		return err
	}

	if sb.handleOnlyAddressAnnotations(is, ca, sa) {
		return nil
	}

	if sa != nil {
		sb.handleSAPresent(is, sa)
	}

	if ca != nil {
		sb.handleCAPresent(is, ca)
	}

	if ma != nil {
		sb.handleMAPresent(is, ma)
	}
	return nil
}

func (sb *spanBuilder) pullOutSpecialBinaryAnnotations(is *InputSpan) (*trace.Endpoint, *trace.Endpoint, *trace.Endpoint, error) {
	var ca, sa, ma *trace.Endpoint
	for i := range is.BinaryAnnotations {
		ba := is.BinaryAnnotations[i]
		if ba.Value == nil || ba.Key == nil {
			continue
		}
		switch val := (*ba.Value).(type) {
		case bool:
			if *ba.Key == "ca" {
				ca = ba.Endpoint
			} else if *ba.Key == "sa" {
				sa = ba.Endpoint
			} else if *ba.Key == "ma" {
				ma = ba.Endpoint
			} else {
				tagVal := "false"
				if val {
					tagVal = "true"
				}
				sb.spanForEndpoint(is, ba.Endpoint).Tags[*ba.Key] = tagVal
			}
			continue
		}

		currentSpan := sb.spanForEndpoint(is, ba.Endpoint)
		if err := sb.convertToTagOnSpan(currentSpan, ba); err != nil {
			return nil, nil, nil, err
		}
	}
	return ca, sa, ma, nil
}

func (sb *spanBuilder) convertToTagOnSpan(currentSpan *trace.Span, ba *signalfxformat.BinaryAnnotation) error {
	switch val := (*ba.Value).(type) {
	case string:
		// don't add marker "lc" tags
		if *ba.Key == "lc" && len(val) == 0 {
			return nil
		}
		currentSpan.Tags[*ba.Key] = val
	case []byte:
		currentSpan.Tags[*ba.Key] = string(val)
	case float64:
		currentSpan.Tags[*ba.Key] = strconv.FormatFloat(val, 'f', -1, 64)
	case int8, int16, int32, int64, uint8, uint16, uint32, uint64:
		currentSpan.Tags[*ba.Key] = fmt.Sprintf("%d", val)
	default:
		fmt.Printf("invalid binary annotation type of %s, for key %s for span %s\n", reflect.TypeOf(val), *ba.Key, *currentSpan.Name)
		return fmt.Errorf("invalid binary annotation type of %s, for key %s", reflect.TypeOf(val), *ba.Key)
	}
	return nil
}

// special-case when we are missing core annotations, but we have both address annotations
func (sb *spanBuilder) handleOnlyAddressAnnotations(is *InputSpan, ca, sa *trace.Endpoint) bool {
	if sb.cs == nil && sb.sr == nil && ca != nil && sa != nil {
		sb.spanForEndpoint(is, ca).RemoteEndpoint = sa
		return true
	}
	return false
}

func (sb *spanBuilder) handleSAPresent(is *InputSpan, sa *trace.Endpoint) {
	if sb.cs != nil && !closeEnough(sa, sb.cs.Endpoint) {
		sb.spanForEndpoint(is, sb.cs.Endpoint).RemoteEndpoint = sa
	} else if sb.cr != nil && !closeEnough(sa, sb.cr.Endpoint) {
		sb.spanForEndpoint(is, sb.cr.Endpoint).RemoteEndpoint = sa
	} else if sb.cs == nil && sb.cr == nil && sb.sr == nil && sb.ss == nil { // no core annotations
		s := sb.spanForEndpoint(is, nil)
		s.Kind = &ClientKind
		s.RemoteEndpoint = sa
	}
}

func (sb *spanBuilder) handleCAPresent(is *InputSpan, ca *trace.Endpoint) {
	if sb.sr != nil && !closeEnough(ca, sb.sr.Endpoint) {
		sb.spanForEndpoint(is, sb.sr.Endpoint).RemoteEndpoint = ca
	}
	if sb.ss != nil && !closeEnough(ca, sb.ss.Endpoint) {
		sb.spanForEndpoint(is, sb.ss.Endpoint).RemoteEndpoint = ca
	} else if sb.cs == nil && sb.cr == nil && sb.sr == nil && sb.ss == nil { // no core annotations
		s := sb.spanForEndpoint(is, nil)
		s.Kind = &ServerKind
		s.RemoteEndpoint = ca
	}
}

func (sb *spanBuilder) handleMAPresent(is *InputSpan, ma *trace.Endpoint) {
	if sb.ms != nil && !closeEnough(ma, sb.ms.Endpoint) {
		sb.spanForEndpoint(is, sb.ms.Endpoint).RemoteEndpoint = ma
	}
	if sb.mr != nil && !closeEnough(ma, sb.mr.Endpoint) {
		sb.spanForEndpoint(is, sb.mr.Endpoint).RemoteEndpoint = ma
	}
}

func closeEnough(left, right *trace.Endpoint) bool {
	if left.ServiceName == nil || right.ServiceName == nil {
		return left.ServiceName == nil && right.ServiceName == nil
	}
	return *left.ServiceName == *right.ServiceName
}

// An error wrapper that is nil-safe and requires no initialization.
type traceErrs struct {
	count   int
	lastErr error
}

// ToError returns the err object if the it has not been instantiated, and itself if it has
// we do this because err is possibly a response from sbingest which could be a json response
// and we want to pass this on unmolested but encoding errors are more important so send them
// if they exist
func (te *traceErrs) ToError(err error) error {
	if te == nil {
		return err
	}
	return te
}

func (te *traceErrs) Error() string {
	return fmt.Sprintf("%d errors encountered, last one was: %s", te.count, te.lastErr.Error())
}

func (te *traceErrs) Append(err error) *traceErrs {
	if err == nil {
		return te
	}

	out := te
	if out == nil {
		out = &traceErrs{}
	}
	out.count++
	out.lastErr = err

	return out
}

// A parentSpanID of all hex 0s should be normalized to nil.
func normalizeParentSpanID(parentSpanID *string) *string {
	if parentSpanID != nil && strings.Count(*parentSpanID, "0") == len(*parentSpanID) {
		return nil
	}
	return parentSpanID
}

// JSONTraceDecoderV1 decodes json to structs
type JSONTraceDecoderV1 struct {
	Logger log.Logger
	Sink   trace.Sink
}

var errInvalidJSONTraceFormat = errors.New("invalid JSON format; please see correct format at https://zipkin.io/zipkin-api/#/default/post_spans")

// Read the data off the wire in json format
func (decoder *JSONTraceDecoderV1) Read(ctx context.Context, req *http.Request) error {
	var input signalfxformat.InputSpanList
	if err := easyjson.UnmarshalFromReader(req.Body, &input); err != nil {
		return errInvalidJSONTraceFormat
	}

	if len(input) == 0 {
		return nil
	}

	spans := make([]*trace.Span, 0, len(input))

	// Don't let an error converting one set of spans prevent other valid spans
	// in the same request from being rejected.
	var conversionErrs *traceErrs
	for _, is := range input {
		inputSpan := (*InputSpan)(is)
		if inputSpan.isDefinitelyZipkinV2() {
			s, err := inputSpan.fromZipkinV2()
			if err != nil {
				conversionErrs = conversionErrs.Append(err)
				continue
			}

			spans = append(spans, s)
		} else {
			derived, err := inputSpan.fromZipkinV1()
			if err != nil {
				conversionErrs = conversionErrs.Append(err)
				continue
			}

			// Zipkin v1 spans can map to multiple spans in Zipkin v2
			spans = append(spans, derived...)
		}
	}

	err := decoder.Sink.AddSpans(ctx, spans)
	return conversionErrs.ToError(err)
}

func setupJSONTraceV1(ctx context.Context, r *mux.Router, sink Sink, logger log.Logger, httpChain web.NextConstructor) sfxclient.Collector {
	handler, st := SetupChain(ctx, sink, ZipkinV1, func(s Sink) ErrorReader {
		return &JSONTraceDecoderV1{Logger: logger, Sink: sink}
	}, httpChain, logger)
	SetupJSONByPaths(r, handler, DefaultTracePathV1)
	return st
}
