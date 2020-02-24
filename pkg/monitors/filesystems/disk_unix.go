// Duplication of github.com/shirou/gopsutil/disk/disk_unix.go to gain access to golang.org/x/sys/unix.Statfs_t.

package filesystems

import (
	"github.com/shirou/gopsutil/disk"
	"golang.org/x/sys/unix"
)

type diskUsageStat struct {
	disk.UsageStat
	Reserved             uint64  `json:"reserved"`
	BytesReservedPercent float64 `json:"bytesReservedPercent"`
	// TODO: add fields InodesReserved and InodesReservedPercent
	// Derive how InodesReserved is calculated from https://github.com/collectd/collectd/blob/master/src/df.c#L303.
	// However, we are using Statfs_t and Statfs_t does not expose available inodes.
}

// Usage returns a file system usage. path is a filesystem path such
// as "/", not device file path like "/dev/vda1". If you want to use
// a return value of disk.Partitions, use "Mountpoint" not "Device".
func diskUsage(path string) (*diskUsageStat, error) {
	stat := unix.Statfs_t{}
	err := unix.Statfs(path, &stat)
	if err != nil {
		return nil, err
	}
	return newDiskUsageStat(&stat)
}

func newDiskUsageStat(stat *unix.Statfs_t) (*diskUsageStat, error) {
	bsize := stat.Bsize
	ret := &diskUsageStat{
		UsageStat: disk.UsageStat{
			Total:       stat.Blocks * uint64(bsize),
			Free:        stat.Bavail * uint64(bsize),
			InodesTotal: stat.Files,
			InodesFree:  stat.Ffree,
		},
	}

	// if could not get InodesTotal, return empty
	if ret.InodesTotal < ret.InodesFree {
		return ret, nil
	}

	ret.InodesUsed = ret.InodesTotal - ret.InodesFree
	ret.Used = (stat.Blocks - stat.Bfree) * uint64(bsize)

	if ret.InodesTotal > 0 {
		ret.InodesUsedPercent = (100.0 * float64(ret.InodesUsed)) / float64(ret.InodesTotal)
	}

	if usedAndFree := ret.Used + ret.Free; usedAndFree > 0 {
		// We don't use ret.Total to calculate percent. see https://github.com/shirou/gopsutil/issues/562
		ret.UsedPercent = (100.0 * float64(ret.Used)) / float64(usedAndFree)
	}

	bReserved := stat.Bfree - stat.Bavail
	ret.Reserved = bReserved * uint64(bsize)
	ret.BytesReservedPercent = (100.0 * float64(bReserved)) / float64(stat.Blocks)

	return ret, nil
}
