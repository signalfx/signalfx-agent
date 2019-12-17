package batchemitter

import (
	"reflect"
	"testing"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/metric"
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/event"
	"github.com/signalfx/signalfx-agent/pkg/neotest"
	log "github.com/sirupsen/logrus"
)

func TestImmediateEmitter_Emit(t *testing.T) {
	type args struct {
		measurement  string
		fields       map[string]interface{}
		tags         map[string]string
		metricType   telegraf.ValueType
		t            time.Time
		includeEvent []string
		excludeData  []string
	}
	ts := time.Now()
	tests := []struct {
		name           string
		args           args
		wantDatapoints []*datapoint.Datapoint
		wantEvents     []*event.Event
	}{
		{
			name: "emit datapoint without plugin tag",
			args: args{
				measurement: "name",
				fields: map[string]interface{}{
					"fieldname": 5,
				},
				tags: map[string]string{
					"dim1Key": "dim1Val",
				},
				metricType: telegraf.Gauge,
				t:          ts,
			},
			wantDatapoints: []*datapoint.Datapoint{
				datapoint.New(
					"name.fieldname",
					map[string]string{
						"dim1Key": "dim1Val",
						"plugin":  "name",
					},
					datapoint.NewIntValue(int64(5)),
					datapoint.Gauge,
					ts),
			},
		},
	}
	for _, tt := range tests {
		args := tt.args
		wantDatapoints := tt.wantDatapoints
		wantEvents := tt.wantEvents

		t.Run(tt.name, func(t *testing.T) {
			out := neotest.NewTestOutput()
			lg := log.NewEntry(log.New())
			I := NewEmitter(out, lg)
			I.IncludeEvents(args.includeEvent)
			I.ExcludeData(args.excludeData)
			m, _ := metric.New(args.measurement, args.tags, args.fields, args.t, args.metricType)
			I.AddMetric(m)
			I.Send()

			dps := out.FlushDatapoints()
			if !reflect.DeepEqual(dps, wantDatapoints) {
				t.Errorf("actual output: datapoints %v does not match desired: %v", dps, wantDatapoints)
			}

			events := out.FlushEvents()
			if !reflect.DeepEqual(events, wantEvents) {
				t.Errorf("actual output: events %v does not match desired: %v", dps, wantDatapoints)
			}
		})
	}
}
