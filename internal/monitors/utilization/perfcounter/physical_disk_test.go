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

func TestPhysicalDisk(t *testing.T) {
	var monitorType = "system-utilization"
	var Measurement = "win_physical_disk"
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
			name: "disk.read_ops.per_second",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields: map[string]interface{}{
							"Disk_Reads_persec": 80,
						},
						Tags: map[string]string{
							"instance": "C:",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf: PhysicalDisk(),
			want: []*datapoint.Datapoint{
				datapoint.New("disk.read_ops.per_second", map[string]string{"plugin": monitorType, "plugin_instance": "C:"}, datapoint.NewIntValue(80), datapoint.Gauge, time.Time{}),
			},
			wantErrors: nil,
		},
		{
			name: "disk.write_ops.per_second",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields: map[string]interface{}{
							"Disk_Writes_persec": 90,
						},
						Tags: map[string]string{
							"instance": "D:",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf: PhysicalDisk(),
			want: []*datapoint.Datapoint{
				datapoint.New("disk.write_ops.per_second", map[string]string{"plugin": monitorType, "plugin_instance": "D:"}, datapoint.NewIntValue(90), datapoint.Gauge, time.Time{}),
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
							"instance": "C:",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf:       PhysicalDisk(),
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
			perf:       PhysicalDisk(),
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
							"Disk_Reads_persec": "Foo",
						},
						Tags: map[string]string{
							"instance": "C:",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf:       PhysicalDisk(),
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
							"Disk_Writes_persec": 90,
						},
						Tags: map[string]string{
							"noinstance": "0",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf:       PhysicalDisk(),
			want:       nil,
			wantErrors: []error{fmt.Errorf("no instance tag defined in tags 'map[noinstance:0]' for measurement '%s'", Measurement)},
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
