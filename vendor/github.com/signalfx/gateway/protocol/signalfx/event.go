package signalfx

import (
	"bytes"
	"context"
	"net/http"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/mailru/easyjson"
	"github.com/signalfx/com_signalfx_metrics_protobuf"
	"github.com/signalfx/gateway/protocol/signalfx/format"
	"github.com/signalfx/golib/datapoint/dpsink"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/golib/log"
	"github.com/signalfx/golib/pointer"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/golib/web"
)

// ProtobufEventDecoderV2 decodes protocol buffers in signalfx's v2 format and sends them to Sink
type ProtobufEventDecoderV2 struct {
	Sink   dpsink.ESink
	Logger log.Logger
}

func (decoder *ProtobufEventDecoderV2) Read(ctx context.Context, req *http.Request) (err error) {
	jeff := buffs.Get().(*bytes.Buffer)
	defer buffs.Put(jeff)
	jeff.Reset()
	if err = readFromRequest(jeff, req, decoder.Logger); err != nil {
		return err
	}
	var msg com_signalfx_metrics_protobuf.EventUploadMessage
	if err = proto.Unmarshal(jeff.Bytes(), &msg); err != nil {
		return err
	}
	evts := make([]*event.Event, 0, len(msg.GetEvents()))
	for _, protoDb := range msg.GetEvents() {
		if e, err := NewProtobufEvent(protoDb); err == nil {
			evts = append(evts, e)
		}
	}
	if len(evts) > 0 {
		err = decoder.Sink.AddEvents(ctx, evts)
	}
	return err
}

// JSONEventDecoderV2 decodes v2 json data for signalfx events and sends it to Sink
type JSONEventDecoderV2 struct {
	Sink   dpsink.ESink
	Logger log.Logger
}

func (decoder *JSONEventDecoderV2) Read(ctx context.Context, req *http.Request) error {
	var e signalfxformat.JSONEventV2
	if err := easyjson.UnmarshalFromReader(req.Body, &e); err != nil {
		return err
	}
	evts := make([]*event.Event, 0, len(e))
	for _, jsonEvent := range e {
		if jsonEvent.Category == nil {
			jsonEvent.Category = pointer.String("USER_DEFINED")
		}
		if jsonEvent.Timestamp == nil {
			jsonEvent.Timestamp = pointer.Int64(0)
		}
		cat := event.USERDEFINED
		if pbcat, ok := com_signalfx_metrics_protobuf.EventCategory_value[*jsonEvent.Category]; ok {
			cat = event.Category(pbcat)
		}
		evt := event.NewWithProperties(jsonEvent.EventType, cat, jsonEvent.Dimensions, jsonEvent.Properties, fromTs(*jsonEvent.Timestamp))
		evts = append(evts, evt)
	}
	return decoder.Sink.AddEvents(ctx, evts)
}

func setupJSONEventV2(ctx context.Context, r *mux.Router, sink Sink, logger log.Logger, debugContext *web.HeaderCtxFlag, httpChain web.NextConstructor) sfxclient.Collector {
	additionalConstructors := []web.Constructor{}
	if debugContext != nil {
		additionalConstructors = append(additionalConstructors, debugContext)
	}
	handler, st := SetupChain(ctx, sink, "json_event_v2", func(s Sink) ErrorReader {
		return &JSONEventDecoderV2{Sink: s, Logger: logger}
	}, httpChain, logger, additionalConstructors...)
	SetupJSONV2EventPaths(r, handler)
	return st
}

// SetupJSONV2EventPaths tells the router which paths the given handler (which should handle v2 protobufs)
func SetupJSONV2EventPaths(r *mux.Router, handler http.Handler) {
	SetupJSONByPaths(r, handler, "/v2/event")
}

func setupProtobufEventV2(ctx context.Context, r *mux.Router, sink Sink, logger log.Logger, debugContext *web.HeaderCtxFlag, httpChain web.NextConstructor) sfxclient.Collector {
	additionalConstructors := []web.Constructor{}
	if debugContext != nil {
		additionalConstructors = append(additionalConstructors, debugContext)
	}
	handler, st := SetupChain(ctx, sink, "protobuf_event_v2", func(s Sink) ErrorReader {
		return &ProtobufEventDecoderV2{Sink: s, Logger: logger}
	}, httpChain, logger, additionalConstructors...)
	SetupProtobufV2EventPaths(r, handler)

	return st
}

// SetupProtobufV2EventPaths tells the router which paths the given handler (which should handle v2 protobufs)
func SetupProtobufV2EventPaths(r *mux.Router, handler http.Handler) {
	SetupProtobufV2ByPaths(r, handler, "/v2/event")
}
