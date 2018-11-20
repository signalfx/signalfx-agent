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
		})
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
		})
		assert.False(t, f.Matches(&datapoint.Datapoint{Metric: "cpu.utilization"}))
		assert.False(t, f.Matches(&datapoint.Datapoint{Metric: "memory.utilization"}))
		assert.False(t, f.Matches(&datapoint.Datapoint{Metric: "disk.utilization"}))
		assert.True(t, f.Matches(&datapoint.Datapoint{Metric: "other.utilization"}))
	})
}
