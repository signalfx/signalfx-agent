// +build windows

package processlist

import (
	"reflect"
	"testing"
	"time"

	"github.com/signalfx/golib/v3/event"
	"github.com/signalfx/golib/v3/pointer"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/neotest"
)

func TestMonitor_Configure(t *testing.T) {
	tests := []struct {
		name       string
		m          *Monitor
		processes  []Win32Process
		cpuPercent map[uint32]uint64
		usernames  map[uint32]string
		want       *event.Event
		wantErr    bool
	}{
		{
			name: "test1",
			m:    &Monitor{Output: neotest.NewTestOutput()},
			processes: []Win32Process{
				{
					Name:           "testProcess1",
					ExecutablePath: pointer.String("C:\\HelloWorld.exe"),
					CommandLine:    pointer.String("HelloWorld.exe"),
					Priority:       8,
					ProcessID:      0,
					Status:         pointer.String(""),
					ExecutionState: pointer.Uint16(0),
					KernelModeTime: 1500,
					PageFileUsage:  1600,
					UserModeTime:   1700,
					WorkingSetSize: 1800,
					VirtualSize:    1900,
				},
				{
					Name:           "testProcess2",
					ExecutablePath: pointer.String("C:\\HelloWorld2.exe"),
					CommandLine:    pointer.String("HelloWorld2.exe"),
					Priority:       8,
					ProcessID:      1,
					Status:         pointer.String(""),
					ExecutionState: pointer.Uint16(0),
					KernelModeTime: 1500,
					PageFileUsage:  1600,
					UserModeTime:   1700,
					WorkingSetSize: 1800,
					VirtualSize:    1900,
				},
			},
			usernames: map[uint32]string{
				0: "tedMosby",
				1: "barneyStinson",
			},
			want: &event.Event{
				EventType:  "objects.top-info",
				Category:   event.AGENT,
				Dimensions: map[string]string{},
				Properties: map[string]interface{}{
					"message": "{\"t\":\"eJyqVjJQsopWKklN8c0vTqpU0rHQUSrNy87LL89T0jHUMdQx0FFS0jHQMzCAEEoGBlYGIJaSjpKzVUyMR2pOTn54flFOil5qRapSrI6SIci8pMSivNTK4JLMvOL8PAoMNYKYWgsIAAD//+q6LfA=\",\"v\":\"0.0.30\"}",
				},
			},
		},
	}
	for i := range tests {
		origGetAllProcesses := getAllProcesses
		origGetUsername := getUsername

		tt := tests[i]

		t.Run(tt.name, func(t *testing.T) {
			getAllProcesses = func() ([]Win32Process, error) {
				return tt.processes, nil
			}
			getUsername = func(id uint32) (string, error) {
				username, ok := tt.usernames[id]
				if !ok {
					t.Error("unable to find username")
				}
				return username, nil
			}
			if err := tt.m.Configure(&Config{config.MonitorConfig{IntervalSeconds: 10}}); (err != nil) != tt.wantErr {
				t.Errorf("Monitor.Configure() error = %v, wantErr %v", err, tt.wantErr)
			}
			time.Sleep(3 * time.Second)
			events := tt.m.Output.(*neotest.TestOutput).FlushEvents()
			if len(events) == 0 {
				t.Errorf("events %v != %v", events, tt.want)
				return
			}

			lastEvent := events[len(events)-1]

			w := tt.want
			if lastEvent.EventType != w.EventType ||
				lastEvent.Category != w.Category ||
				!reflect.DeepEqual(lastEvent.Dimensions, w.Dimensions) ||
				!reflect.DeepEqual(lastEvent.Properties, w.Properties) {
				t.Errorf("events %v != %v", lastEvent, tt.want)
				return
			}
		})
		getAllProcesses = origGetAllProcesses
		getUsername = origGetUsername
	}
}
