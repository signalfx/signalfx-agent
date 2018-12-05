package config

import (
	"testing"

	"github.com/signalfx/golib/datapoint"
	"github.com/stretchr/testify/assert"
)

func TestFilters(t *testing.T) {
	t.Run("Make single filter properly", func(t *testing.T) {
		f, _ := makeFilterSet([]MetricFilter{
			MetricFilter{
				MetricNames: []string{
					"cpu.utilization",
					"memory.utilization",
				},
			},
		}, nil)
		assert.True(t, f.Matches(&datapoint.Datapoint{Metric: "cpu.utilization"}))
		assert.True(t, f.Matches(&datapoint.Datapoint{Metric: "memory.utilization"}))
		assert.False(t, f.Matches(&datapoint.Datapoint{Metric: "disk.utilization"}))
	})

	t.Run("Merges two filters properly", func(t *testing.T) {
		f, _ := makeFilterSet([]MetricFilter{
			MetricFilter{
				MetricNames: []string{
					"cpu.utilization",
					"memory.utilization",
				},
				Negated: true,
			},
			MetricFilter{
				MetricNames: []string{
					"disk.utilization",
				},
				Negated: true,
			},
		}, nil)
		assert.False(t, f.Matches(&datapoint.Datapoint{Metric: "cpu.utilization"}))
		assert.False(t, f.Matches(&datapoint.Datapoint{Metric: "memory.utilization"}))
		assert.False(t, f.Matches(&datapoint.Datapoint{Metric: "disk.utilization"}))
		assert.True(t, f.Matches(&datapoint.Datapoint{Metric: "other.utilization"}))
	})

	t.Run("Merges include filters properly", func(t *testing.T) {
		f, _ := makeFilterSet([]MetricFilter{
			MetricFilter{
				MetricNames: []string{
					"cpu.utilization",
					"memory.utilization",
				},
				Negated: true,
			},
			MetricFilter{
				MetricNames: []string{
					"disk.utilization",
				},
				Negated: true,
			},
		}, []MetricFilter{
			MetricFilter{
				MetricNames: []string{
					"my.metric",
				},
			},
		})
		assert.False(t, f.Matches(&datapoint.Datapoint{Metric: "cpu.utilization"}))
		assert.False(t, f.Matches(&datapoint.Datapoint{Metric: "memory.utilization"}))
		assert.False(t, f.Matches(&datapoint.Datapoint{Metric: "disk.utilization"}))
		assert.False(t, f.Matches(&datapoint.Datapoint{Metric: "my.metric"}))
		assert.True(t, f.Matches(&datapoint.Datapoint{Metric: "random.metric"}))
	})

	t.Run("Include filters with dims take priority", func(t *testing.T) {
		f, _ := makeFilterSet([]MetricFilter{
			MetricFilter{
				MetricNames: []string{
					"cpu.utilization",
					"memory.utilization",
				},
			},
			MetricFilter{
				Dimensions: map[string]string{
					"app": "myapp",
				},
			},
		}, []MetricFilter{
			MetricFilter{
				MetricNames: []string{
					"cpu.utilization",
				},
				Dimensions: map[string]string{
					"app": "myapp",
				},
			},
		})
		assert.True(t, f.Matches(&datapoint.Datapoint{Metric: "cpu.utilization"}))
		assert.False(t, f.Matches(&datapoint.Datapoint{Metric: "cpu.utilization", Dimensions: map[string]string{"app": "myapp"}}))
		assert.True(t, f.Matches(&datapoint.Datapoint{Metric: "memory.utilization"}))
		assert.False(t, f.Matches(&datapoint.Datapoint{Metric: "disk.utilization"}))
		assert.False(t, f.Matches(&datapoint.Datapoint{Metric: "random.metric"}))
	})
}
