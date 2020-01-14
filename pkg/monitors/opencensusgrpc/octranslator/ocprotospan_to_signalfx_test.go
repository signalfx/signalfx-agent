package octranslator

import (
	"encoding/json"
	"reflect"
	"testing"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	sfxtrace "github.com/signalfx/golib/trace"
)

var tds = []consumerdata.TraceData{
	{
		Node: &commonpb.Node{
			Identifier: &commonpb.ProcessIdentifier{
				HostName:       "api246-sjc1",
				Pid:            13,
				StartTimestamp: &timestamp.Timestamp{Seconds: 1485467190, Nanos: 639875000},
			},
			LibraryInfo: &commonpb.LibraryInfo{ExporterVersion: "someVersion"},
			ServiceInfo: &commonpb.ServiceInfo{Name: "api"},
			Attributes: map[string]string{
				"a.binary": "AQIDBAMCAQ==",
				"a.bool":   "true",
				"a.double": "1234.56789",
				"a.long":   "123456789",
				"ip":       "10.53.69.61",
			},
		},
		Resource: &resourcepb.Resource{
			Type:   "k8s.io/container",
			Labels: map[string]string{"resource_key1": "resource_val1"},
		},
		Spans: []*tracepb.Span{
			{
				TraceId:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x52, 0x96, 0x9A, 0x89, 0x55, 0x57, 0x1A, 0x3F},
				SpanId:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x7D, 0x98},
				ParentSpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x68, 0xC4, 0xE3},
				Name:         &tracepb.TruncatableString{Value: "get"},
				Kind:         tracepb.Span_CLIENT,
				StartTime:    &timestamp.Timestamp{Seconds: 1485467191, Nanos: 639875000},
				EndTime:      &timestamp.Timestamp{Seconds: 1485467191, Nanos: 662813000},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"http.url": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "http://localhost:15598/client_transactions"}},
						},
						"peer.ipv4": {
							Value: &tracepb.AttributeValue_IntValue{IntValue: 3224716605},
						},
						"peer.port": {
							Value: &tracepb.AttributeValue_IntValue{IntValue: 53931},
						},
						"peer.service": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "rtapi"}},
						},
						"someBool": {
							Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
						},
						"someDouble": {
							Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: 129.8},
						},
						"span.kind": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "client"}},
						},
					},
				},
				TimeEvents: &tracepb.Span_TimeEvents{
					TimeEvent: []*tracepb.Span_TimeEvent{
						{
							Time: &timestamp.Timestamp{Seconds: 1485467191, Nanos: 639874000},
							Value: &tracepb.Span_TimeEvent_MessageEvent_{
								MessageEvent: &tracepb.Span_TimeEvent_MessageEvent{
									Type: tracepb.Span_TimeEvent_MessageEvent_SENT, UncompressedSize: 1024, CompressedSize: 512,
								},
							},
						},
						{
							Time: &timestamp.Timestamp{Seconds: 1485467191, Nanos: 639875000},
							Value: &tracepb.Span_TimeEvent_Annotation_{
								Annotation: &tracepb.Span_TimeEvent_Annotation{
									Attributes: &tracepb.Span_Attributes{
										AttributeMap: map[string]*tracepb.AttributeValue{
											"key1": {
												Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "value1"}},
											},
										},
									},
								},
							},
						},
						{
							Time: &timestamp.Timestamp{Seconds: 1485467191, Nanos: 639875000},
							Value: &tracepb.Span_TimeEvent_Annotation_{
								Annotation: &tracepb.Span_TimeEvent_Annotation{
									Description: &tracepb.TruncatableString{Value: "annotation description"},
									Attributes: &tracepb.Span_Attributes{
										AttributeMap: map[string]*tracepb.AttributeValue{
											"event": {
												Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "nothing"}},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},
	//{
	//	Node: &commonpb.Node{
	//		ServiceInfo: &commonpb.ServiceInfo{Name: "api"},
	//	},
	//	Spans: []*tracepb.Span{
	//		{
	//			TraceId:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x52, 0x96, 0x9A, 0x89, 0x55, 0x57, 0x1A, 0x3F},
	//			SpanId:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x7D, 0x98},
	//			ParentSpanId: nil,
	//			Name:         &tracepb.TruncatableString{Value: "get"},
	//			Kind:         tracepb.Span_SERVER,
	//			StartTime:    &timestamp.Timestamp{Seconds: 1485467191, Nanos: 639875000},
	//			EndTime:      &timestamp.Timestamp{Seconds: 1485467191, Nanos: 662813000},
	//			Attributes: &tracepb.Span_Attributes{
	//				AttributeMap: map[string]*tracepb.AttributeValue{
	//					"peer.service": {
	//						Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "AAAAAAAAMDk="}},
	//					},
	//				},
	//			},
	//		},
	//		{
	//			TraceId:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x52, 0x96, 0x9A, 0x89, 0x55, 0x57, 0x1A, 0x3F},
	//			SpanId:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x7D, 0x99},
	//			ParentSpanId: []byte{},
	//			Name:         &tracepb.TruncatableString{Value: "get"},
	//			Kind:         tracepb.Span_SERVER,
	//			StartTime:    &timestamp.Timestamp{Seconds: 1485467191, Nanos: 639875000},
	//			EndTime:      &timestamp.Timestamp{Seconds: 1485467191, Nanos: 662813000},
	//			Links: &tracepb.Span_Links{
	//				Link: []*tracepb.Span_Link{
	//					{
	//						TraceId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x52, 0x96, 0x9A, 0x89, 0x55, 0x57, 0x1A, 0x3F},
	//						SpanId:  []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x7D, 0x98},
	//						Type:    tracepb.Span_Link_PARENT_LINKED_SPAN,
	//					},
	//					{
	//						TraceId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x52, 0x96, 0x9A, 0x89, 0x55, 0x57, 0x1A, 0x3F},
	//						SpanId:  []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x68, 0xC4, 0xE3},
	//					},
	//				},
	//			},
	//		},
	//		{
	//			TraceId:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x52, 0x96, 0x9A, 0x89, 0x55, 0x57, 0x1A, 0x3F},
	//			SpanId:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x7D, 0x98},
	//			ParentSpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	//			Name:         &tracepb.TruncatableString{Value: "get2"},
	//			StartTime:    &timestamp.Timestamp{Seconds: 1485467192, Nanos: 639875000},
	//			EndTime:      &timestamp.Timestamp{Seconds: 1485467192, Nanos: 662813000},
	//		},
	//	},
	//},
}

//func TestOCProtoSpanToSignalFx1(t *testing.T) {
//	type args struct {
//		serviceName string
//		s           *tracepb.Span
//	}
//	tests := []struct {
//		name string
//		args args
//		want *sfxtrace.Span
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := OCProtoSpanToSignalFx(tt.args.serviceName, tt.args.s); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("OCProtoSpanToSignalFx() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}

func TestOCProtoSpansToSignalFx(t *testing.T) {
	type args struct {
		td consumerdata.TraceData
	}
	tests := []struct {
		name string
		args args
		want []*sfxtrace.Span
	}{
		{
			name: "",
			args: args{
				td: tds[0],
			},
			want: []*sfxtrace.Span{

			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OCProtoSpansToSignalFx(tt.args.td); !reflect.DeepEqual(got, tt.want) {

				bts, _ := json.Marshal(got)
				t.Log(string(bts))
				t.Errorf("OCProtoSpansToSignalFx() = %v, want %v", got, tt.want)
			}
		})
	}
}
//
//func Test_attributeValueToString(t *testing.T) {
//	type args struct {
//		attr *tracepb.AttributeValue
//	}
//	tests := []struct {
//		name  string
//		args  args
//		want  string
//		want1 bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, got1 := attributeValueToString(tt.args.attr)
//			if got != tt.want {
//				t.Errorf("attributeValueToString() got = %v, want %v", got, tt.want)
//			}
//			if got1 != tt.want1 {
//				t.Errorf("attributeValueToString() got1 = %v, want %v", got1, tt.want1)
//			}
//		})
//	}
//}
//
//func Test_attributesToTags(t *testing.T) {
//	type args struct {
//		redundantKeys map[string]bool
//		attrMap       map[string]*tracepb.AttributeValue
//	}
//	tests := []struct {
//		name string
//		args args
//		want map[string]string
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := attributesToTags(tt.args.redundantKeys, tt.args.attrMap); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("attributesToTags() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func Test_convertSpanID(t *testing.T) {
//	type args struct {
//		s []byte
//	}
//	tests := []struct {
//		name string
//		args args
//		want string
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := convertSpanID(tt.args.s); got != tt.want {
//				t.Errorf("convertSpanID() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func Test_convertTraceID(t *testing.T) {
//	type args struct {
//		t []byte
//	}
//	tests := []struct {
//		name string
//		args args
//		want string
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := convertTraceID(tt.args.t); got != tt.want {
//				t.Errorf("convertTraceID() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func Test_getDurationInMicrosecondsFromTimestamps(t *testing.T) {
//	type args struct {
//		start *timestamp.Timestamp
//		end   *timestamp.Timestamp
//	}
//	tests := []struct {
//		name string
//		args args
//		want *int64
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := getDurationInMicrosecondsFromTimestamps(tt.args.start, tt.args.end); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("getDurationInMicrosecondsFromTimestamps() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func Test_getEndpointFromAttributes(t *testing.T) {
//	type args struct {
//		attributes    *tracepb.Span_Attributes
//		serviceName   string
//		redundantKeys map[string]bool
//		ipv4Key       string
//		ipv6Key       string
//		portKey       string
//	}
//	tests := []struct {
//		name string
//		args args
//		want *sfxtrace.Endpoint
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := getEndpointFromAttributes(tt.args.attributes, tt.args.serviceName, tt.args.redundantKeys, tt.args.ipv4Key, tt.args.ipv6Key, tt.args.portKey); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("getEndpointFromAttributes() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func Test_getStringAttribute(t *testing.T) {
//	type args struct {
//		attributes map[string]*tracepb.AttributeValue
//		key        string
//	}
//	tests := []struct {
//		name      string
//		args      args
//		wantValue string
//		wantOk    bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			gotValue, gotOk := getStringAttribute(tt.args.attributes, tt.args.key)
//			if gotValue != tt.wantValue {
//				t.Errorf("getStringAttribute() gotValue = %v, want %v", gotValue, tt.wantValue)
//			}
//			if gotOk != tt.wantOk {
//				t.Errorf("getStringAttribute() gotOk = %v, want %v", gotOk, tt.wantOk)
//			}
//		})
//	}
//}
//
//func Test_spanKindToString(t *testing.T) {
//	type args struct {
//		s tracepb.Span_SpanKind
//	}
//	tests := []struct {
//		name string
//		args args
//		want *string
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := spanKindToString(tt.args.s); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("spanKindToString() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func Test_spanTimeEventMessageEventTypeToString(t *testing.T) {
//	type args struct {
//		t tracepb.Span_TimeEvent_MessageEvent_Type
//	}
//	tests := []struct {
//		name string
//		args args
//		want *string
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := spanTimeEventMessageEventTypeToString(tt.args.t); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("spanTimeEventMessageEventTypeToString() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func Test_timeEventsToAnnotations(t *testing.T) {
//	type args struct {
//		tes *tracepb.Span_TimeEvents
//	}
//	tests := []struct {
//		name string
//		args args
//		want []*sfxtrace.Annotation
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := timeEventsToAnnotations(tt.args.tes); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("timeEventsToAnnotations() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func Test_timestampToMicroseconds(t *testing.T) {
//	type args struct {
//		ts *timestamp.Timestamp
//	}
//	tests := []struct {
//		name string
//		args args
//		want *int64
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := timestampToMicroseconds(tt.args.ts); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("timestampToMicroseconds() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func Test_truncatableStringToString(t *testing.T) {
//	type args struct {
//		ts *tracepb.TruncatableString
//	}
//	tests := []struct {
//		name string
//		args args
//		want *string
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := truncatableStringToString(tt.args.ts); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("truncatableStringToString() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
