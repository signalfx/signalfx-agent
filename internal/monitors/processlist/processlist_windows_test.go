// +build windows

package processlist

import (
	"reflect"
	"testing"
	"time"

	"github.com/signalfx/golib/event"
	"github.com/signalfx/golib/pointer"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/neotest"
)

func TestMonitor_Configure(t *testing.T) {
	tests := []struct {
		name       string
		m          *Monitor
		processes  []Win32Process
		cpuPercent map[uint32]uint64
		usernames  map[uint32]string
		want       []*event.Event
		wantErr    bool
	}{
		{
			name: "test1",
			m:    &Monitor{Output: neotest.NewTestOutput()},
			processes: []Win32Process{
				Win32Process{
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
				Win32Process{
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
			cpuPercent: map[uint32]uint64{
				0: 99,
				1: 98,
			},
			usernames: map[uint32]string{
				0: "tedMosby",
				1: "barneyStinson",
			},
			want: []*event.Event{
				&event.Event{
					EventType:  "objects.top-info",
					Category:   event.AGENT,
					Dimensions: map[string]string{},
					Properties: map[string]interface{}{
						"message": "{\"t\":\"eJyqVjJQsopWKklN8c0vTqpU0rHQUVLSMdQx1DEAMSwt9QwMdAxAhJKBgZUBiKWko+RsFRPjkZqTkx+eX5STopdakaoUq6NkCDIpKbEoL7UyuCQzrzg/D8M4CyKMM4KYVwsIAAD//+vVKFM=\",\"v\":\"0.0.30\"}",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		origGetAllProcesses := getAllProcesses
		origGetCPUPercentages := getCPUPercentages
		origGetUsername := getUsername
		t.Run(tt.name, func(t *testing.T) {
			getAllProcesses = func() ([]Win32Process, error) {
				return tt.processes, nil
			}
			getCPUPercentages = func() (cpuPercents map[uint32]uint64, err error) {
				return tt.cpuPercent, nil
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
			time.Sleep(1 * time.Second)
			events := tt.m.Output.(*neotest.TestOutput).FlushEvents()
			if !reflect.DeepEqual(events, tt.want) {
				t.Errorf("events %v != %v", events, tt.want)
			}
		})
		getAllProcesses = origGetAllProcesses
		getCPUPercentages = origGetCPUPercentages
		getUsername = origGetUsername
	}
}
