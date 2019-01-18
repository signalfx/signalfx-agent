package perfcounter

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/measurement"
	"github.com/signalfx/signalfx-agent/internal/neotest"
)

func TestMemoryMeasurement(t *testing.T) {
	var monitorType = "system-utilization"
	var Measurement = "win_memory"
	type args struct {
		ms          []*measurement.Measurement
		monitorType string
		output      *neotest.TestOutput
	}
	tests := []struct {
		name       string
		args       args
		perf       PerfCounter
		want       []*datapoint.Datapoint
		wantErrors []error
	}{
		{
			name: "vmpage.swapped_in.per_second",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields: map[string]interface{}{
							"Pages_Input_persec": 80,
						},
						Tags: map[string]string{},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf: Memory(),
			want: []*datapoint.Datapoint{
				datapoint.New("vmpage.swapped_in.per_second", map[string]string{"plugin": monitorType}, datapoint.NewIntValue(80), datapoint.Gauge, time.Time{}),
			},
			wantErrors: nil,
		},
		{
			name: "vmpage.swapped_out.per_second",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields: map[string]interface{}{
							"Pages_Output_persec": 90,
						},
						Tags: map[string]string{},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf: Memory(),
			want: []*datapoint.Datapoint{
				datapoint.New("vmpage.swapped_out.per_second", map[string]string{"plugin": monitorType}, datapoint.NewIntValue(90), datapoint.Gauge, time.Time{}),
			},
			wantErrors: nil,
		},
		{
			name: "vmpage.swapped.per_second",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields: map[string]interface{}{
							"Pages_persec": 90,
						},
						Tags: map[string]string{},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf: Memory(),
			want: []*datapoint.Datapoint{
				datapoint.New("vmpage.swapped.per_second", map[string]string{"plugin": monitorType}, datapoint.NewIntValue(90), datapoint.Gauge, time.Time{}),
			},
			wantErrors: nil,
		},
		{
			name: "bad field",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields: map[string]interface{}{
							"NotARealField": 90,
						},
						Tags: map[string]string{},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf:       Memory(),
			want:       nil,
			wantErrors: []error{fmt.Errorf("unable to map field 'NotARealField' to a metricname while parsing measurement '%s'", Measurement)},
		},
		{
			name: "no fields",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields:      map[string]interface{}{},
						Tags:        map[string]string{},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf:       Memory(),
			want:       nil,
			wantErrors: []error{fmt.Errorf("no fields on measurement '%s'", Measurement)},
		},
		{
			name: "bad value",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields: map[string]interface{}{
							"Pages_Output_persec": "Foo",
						},
						Tags: map[string]string{},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf:       Memory(),
			want:       nil,
			wantErrors: []error{fmt.Errorf("unknown metric value type string")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, ms := range tt.args.ms {
				if gotErrors := tt.perf.ProcessMeasurement(ms, tt.args.monitorType, tt.args.output.SendDatapoint); !reflect.DeepEqual(gotErrors, tt.wantErrors) {
					t.Errorf("ProcessMeasurement() = %v, want %v", gotErrors, tt.wantErrors)
					continue
				}
			}
			if dps := tt.args.output.FlushDatapoints(); !reflect.DeepEqual(tt.want, dps) {
				t.Errorf("ProcessMeasurement() = %v, want %v", dps, tt.want)
			}
		})
	}
}
