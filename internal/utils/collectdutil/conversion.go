package collectdutil

import (
	"strings"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/golib/pointer"
	"github.com/signalfx/metricproxy/protocol/collectd"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

// ConvertWriteFormat will take a collectd value list, create datapoints or
// events from it and inject them to the provided slices as applicable.  The
// slices are accepted as arguments to minimize allocation when this function
// is called back-to-back with many value lists.
func ConvertWriteFormat(f *collectd.JSONWriteFormat, dps *[]*datapoint.Datapoint, events *[]*event.Event) {
	if f.Time != nil && f.Severity != nil && f.Message != nil {
		event := collectd.NewEvent(f, nil)
		*events = append(*events, event)
	} else {
		// The converter below expects dstypes to be lower case
		for i := range f.Dstypes {
			f.Dstypes[i] = pointer.String(strings.ToLower(*f.Dstypes[i]))
		}
		for i := range f.Dsnames {
			if i < len(f.Dstypes) && i < len(f.Values) && f.Values[i] != nil {
				dp := collectd.NewDatapoint(f, uint(i), nil)
				dp.Meta = utils.StringInterfaceMapToAllInterfaceMap(f.Meta)
				*dps = append(*dps, dp)
			}
		}
	}
}
