package testhelpers

import (
    sfxproto "github.com/signalfx/com_signalfx_metrics_protobuf"
)


func ProtoDimensionsToMap(dims []*sfxproto.Dimension) (m map[string]string) {
    m = make(map[string]string)

    for _, d := range dims {
        m[d.GetKey()] = d.GetValue()
    }
    return m
}
