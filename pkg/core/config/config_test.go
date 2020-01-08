package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigPropagation(t *testing.T) {
	t.Run("ProcPath is passed down if not present", func(t *testing.T) {
		c := &Config{
			ProcPath: "/hostfs/proc",
			Monitors: []MonitorConfig{
				{},
				{ProcPath: "/proc"},
			},
		}
		err := c.propagateValuesDown()
		require.Nil(t, err)

		require.Equal(t, c.Monitors[0].ProcPath, "/hostfs/proc")
		require.Equal(t, c.Monitors[1].ProcPath, "/proc")
	})
}
