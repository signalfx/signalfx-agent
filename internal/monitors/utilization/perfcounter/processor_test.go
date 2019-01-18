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

func TestProcessorMeasurement(t *testing.T) {
	var monitorType = "system-utilization"
	var Measurement = "win_cpu"
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
			name: "cpu.utilization",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields: map[string]interface{}{
							"Percent_Processor_Time": 80,
						},
						Tags: map[string]string{
							"instance": "_Total",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf: Processor(),
			want: []*datapoint.Datapoint{
				datapoint.New("cpu.utilization", map[string]string{"plugin": monitorType}, datapoint.NewIntValue(80), datapoint.Gauge, time.Time{}),
			},
			wantErrors: nil,
		},
		{
			name: "cpu.utilization_per_core",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields: map[string]interface{}{
							"Percent_Processor_Time": 90,
						},
						Tags: map[string]string{
							"instance": "0",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf: Processor(),
			want: []*datapoint.Datapoint{
				datapoint.New("cpu.utilization_per_core", map[string]string{"plugin": monitorType, "core": "0"}, datapoint.NewIntValue(90), datapoint.Gauge, time.Time{}),
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
						Tags: map[string]string{
							"instance": "0",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf:       Processor(),
			want:       nil,
			wantErrors: nil,
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
			perf:       Processor(),
			want:       nil,
			wantErrors: []error{fmt.Errorf("no fields on processor measurement '%s'", Measurement)},
		},
		{
			name: "bad value",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields: map[string]interface{}{
							"Percent_Processor_Time": "Foo",
						},
						Tags: map[string]string{
							"instance": "0",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf:       Processor(),
			want:       nil,
			wantErrors: []error{fmt.Errorf("unknown metric value type string")},
		},
		{
			name: "no instance",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields: map[string]interface{}{
							"Percent_Processor_Time": "Foo",
						},
						Tags: map[string]string{
							"noinstance": "0",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf:       Processor(),
			want:       nil,
			wantErrors: []error{fmt.Errorf("no instance tag defined in tags 'map[noinstance:0]' on measurement '%s'", Measurement)},
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
