// +build !windows

package filesystems

import gopsutil "github.com/shirou/gopsutil/disk"

var _ nixPartitions = &Monitor{}

type nixPartitions interface {
	partitionsWrapper
}

func (m *Monitor) getPartitions(all bool) ([]gopsutil.PartitionStat, error) {
	return gopsutil.Partitions(all)
}
