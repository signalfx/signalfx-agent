// +build windows

package filesystems

import (
	"testing"

	gopsutil "github.com/shirou/gopsutil/disk"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAllMounts_ShouldInclude_gopsutil_Mounts(t *testing.T) {
	logger := logrus.WithFields(logrus.Fields{"monitorType": monitorType})

	// Drive and folder mounts.
	allMounts := (&Monitor{logger: logger}).getAllMounts()
	require.NotEmpty(t, allMounts, "failed to find any mount points")

	// Mounts from gopsutil are for drives only.
	gopsutilMounts, err := getGopsutilMounts()
	require.NoError(t, err)

	require.NotEmpty(t, gopsutilMounts, "failed to find any mount points using gopsutil")

	assert.Subset(t, allMounts, gopsutilMounts)
}

func TestNewPartitionStats_SameAs_gopsutil_PartitionStats(t *testing.T) {
	// Partition stats from gopsutil are for drive mounts only.
	gopsutilStatsSlice, err := gopsutil.Partitions(true)
	require.NoError(t, err)

	require.NotEmpty(t, gopsutilStatsSlice, "failed to find any partition stats using gopsutil")

	logger := logrus.WithFields(logrus.Fields{"monitorType": monitorType})
	monitor := Monitor{logger: logger}

	// All drive and folder mounts.
	allMounts := monitor.getAllMounts()

	require.NotEmpty(t, allMounts, "failed to find any mount points")

	var newStats gopsutil.PartitionStat
	for _, gopsutilStats := range gopsutilStatsSlice {
		newStats, err = newPartitionStats(gopsutilStats.Mountpoint)
		require.NoError(t, err)

		assert.Equal(t, gopsutilStats, newStats)
	}
}

func TestGetPartitions_ShouldInclude_gopsutil_PartitionStats(t *testing.T) {
	// Partition stats from gopsutil are for drive mounts only.
	gopsutilStats, err := gopsutil.Partitions(true)
	require.NoError(t, err)

	require.NotEmpty(t, gopsutilStats, "failed to find any partition stats using gopsutil")

	logger := logrus.WithFields(logrus.Fields{"monitorType": monitorType})
	monitor := Monitor{logger: logger}

	var stats []gopsutil.PartitionStat
	// Partition stats for drive and folder mounts.
	stats, err = monitor.getPartitions(true)
	require.NoError(t, err)

	require.NotEmpty(t, stats, "failed to find any partition stats")

	assert.Subset(t, stats, gopsutilStats)
}

func getGopsutilMounts() ([]string, error) {
	partitionsStats, err := gopsutil.Partitions(true)
	if err != nil {
		return nil, err
	}

	mounts := make([]string, 0)
	for _, stats := range partitionsStats {
		mounts = append(mounts, stats.Mountpoint)
	}

	return mounts, nil
}
