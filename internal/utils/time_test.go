package utils

import (
	"context"
	"testing"
	"time"
)

type testMonitor struct {
	executions int
}

func (t *testMonitor) Execute() {
	t.executions++
}

func TestRunOnArrayOfIntervals(t *testing.T) {
	cancelledContext, cancel := context.WithCancel(context.Background())
	cancel()
	type args struct {
		ctx          context.Context
		monitor      *testMonitor
		intervals    []time.Duration
		repeatPolicy RepeatPolicy
		wait         time.Duration
	}
	tests := []struct {
		name       string
		args       args
		comparison func(got int) bool
		want       string
	}{
		{
			name: "test repeat none",
			args: args{
				ctx:          context.Background(),
				monitor:      &testMonitor{},
				intervals:    []time.Duration{10 * time.Millisecond, 10 * time.Millisecond, 10 * time.Millisecond, 10 * time.Millisecond},
				repeatPolicy: RepeatNone,
				wait:         1 * time.Second,
			},
			comparison: func(got int) bool { return got == 4 },
			want:       "equal to 4",
		},
		{
			name: "test repeat last",
			args: args{
				ctx:          context.Background(),
				monitor:      &testMonitor{},
				intervals:    []time.Duration{100 * time.Millisecond, 100 * time.Millisecond, 100 * time.Millisecond, 300 * time.Millisecond},
				repeatPolicy: RepeatLast,
				wait:         1 * time.Second,
			},
			comparison: func(got int) bool { return got > 4 },
			want:       "greater than 4",
		},
		{
			name: "test repeat all",
			args: args{
				ctx:          context.Background(),
				monitor:      &testMonitor{},
				intervals:    []time.Duration{100 * time.Millisecond, 100 * time.Millisecond, 100 * time.Millisecond, 100 * time.Millisecond},
				repeatPolicy: RepeatAll,
				wait:         1 * time.Second,
			},
			comparison: func(got int) bool { return got > 8 },
			want:       "greater than 8",
		},
		{
			name: "test no interval",
			args: args{
				ctx:          context.Background(),
				monitor:      &testMonitor{},
				intervals:    []time.Duration{},
				repeatPolicy: RepeatAll,
				wait:         1 * time.Second,
			},
			comparison: func(got int) bool { return got == 0 },
			want:       "0",
		},
		{
			name: "test closed context",
			args: args{
				ctx:          cancelledContext,
				monitor:      &testMonitor{},
				intervals:    []time.Duration{100 * time.Millisecond, 100 * time.Millisecond, 100 * time.Millisecond, 100 * time.Millisecond},
				repeatPolicy: RepeatAll,
				wait:         1 * time.Second,
			},
			comparison: func(got int) bool { return got == 0 },
			want:       "0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RunOnArrayOfIntervals(tt.args.ctx, tt.args.monitor.Execute, tt.args.intervals, tt.args.repeatPolicy)
			time.Sleep(tt.args.wait)
			if !tt.comparison(tt.args.monitor.executions) {
				t.Errorf("expected nubmer of executions to be %v , but got %v", tt.want, tt.args.monitor.executions)
			}
		})
	}
}
