package utils

import (
	"github.com/golang/protobuf/proto"

	"github.com/signalfx/golib/v3/trace"
	"github.com/signalfx/sapm-proto/translator"
)

func SAPMMarshal(v []*trace.Span) ([]byte, error) {
	msg := translator.SFXToSAPMPostRequest(v)
	return proto.Marshal(msg)
}
