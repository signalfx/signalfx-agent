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

func TestPageFileMeasurement(t *testing.T) {
	var monitorType = "signalfx-system-utilization"
	var Measurement = "win_paging_file"
	var perf = PageFile()
	type args struct {
		ms          []*measurement.Measurement
		monitorType string
		output      *neotest.TestOutput
	}
	tests := []struct {
		name       string
		args       args
		want       []*datapoint.Datapoint
		wantErrors []error
	}{
		{
			name: "paging_file.pct_usage",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields: map[string]interface{}{
							"Percent_Usage": 80,
						},
						Tags: map[string]string{
							"instance": "_Total",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			want: []*datapoint.Datapoint{
				datapoint.New("paging_file.pct_usage", map[string]string{"plugin": monitorType, "instance": "_Total"}, datapoint.NewIntValue(80), datapoint.Gauge, time.Time{}),
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
							"instance": "_Total",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
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
						Tags: map[string]string{
							"instance": "_Total",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
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
							"Percent_Usage": "Foo",
						},
						Tags: map[string]string{
							"instance": "_Total",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			want:       nil,
			wantErrors: []error{fmt.Errorf("unknown metric value type string")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, ms := range tt.args.ms {
				if gotErrors := perf.ProcessMeasurement(ms, tt.args.monitorType, tt.args.output.SendDatapoint); !reflect.DeepEqual(gotErrors, tt.wantErrors) {
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
