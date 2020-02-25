// Duplication of github.com/shirou/gopsutil/disk/disk_unix.go to gain access to golang.org/x/sys/unix.Statfs_t.

package filesystems

import (
	"time"

	"github.com/shirou/gopsutil/disk"
	"github.com/signalfx/golib/v3/datapoint"
	"golang.org/x/sys/unix"
)

type fsStat struct {
	disk.UsageStat
	Reserved             uint64  `json:"reserved"`
	BytesReservedPercent float64 `json:"bytesReservedPercent"`
	// TODO: add fields InodesReserved and InodesReservedPercent
	// Derive how InodesReserved is calculated from https://github.com/collectd/collectd/blob/master/src/df.c#L303.
	// However, we are using Statfs_t and Statfs_t does not expose available inodes.
}

// newFsStat returns a file system usage. path is a filesystem path such as "/", not device file path like
// "/dev/vda1". If you want to use a return value of disk.Partitions, use "Mountpoint" not "Device".
func newFsStat(path string) (*fsStat, error) {
	stat := unix.Statfs_t{}
	err := unix.Statfs(path, &stat)
	if err != nil {
		return nil, err
	}
	bsize := stat.Bsize

	ret := &fsStat{
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

func (fs *fsStat) diskUtilizationDatapoint(dimensions map[string]string) *datapoint.Datapoint {
	return datapoint.New(diskUtilization, dimensions, datapoint.NewFloatValue(fs.UsedPercent), datapoint.Gauge, time.Time{})
}

func (fs *fsStat) makeInodeDatapoints(dimensions map[string]string) []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		datapoint.New(dfInodesFree, dimensions, datapoint.NewIntValue(int64(fs.InodesFree)), datapoint.Gauge, time.Time{}),
		datapoint.New(dfInodesUsed, dimensions, datapoint.NewIntValue(int64(fs.InodesUsed)), datapoint.Gauge, time.Time{}),
		// TODO: implement df_inodes.reserved.
		// However, we get inode stats from Statfs_t and Statfs_t does not expose available inodes. Available inodes are
		// needed to calculated inodes reserved according to https://github.com/collectd/collectd/blob/master/src/df.c#L303 .
		datapoint.New(percentInodesFree, dimensions, datapoint.NewIntValue(int64(100-fs.InodesUsedPercent)), datapoint.Gauge, time.Time{}),
		datapoint.New(percentInodesUsed, dimensions, datapoint.NewIntValue(int64(fs.InodesUsedPercent)), datapoint.Gauge, time.Time{}),
		// TODO: implement percent_inodes.reserved
		// But, Statfs_t does not expose available inodes. See comment above.
	}
}

func (fs *fsStat) makeDFComplex(dimensions map[string]string) []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		datapoint.New(dfComplexFree, dimensions, datapoint.NewIntValue(int64(fs.Free)), datapoint.Gauge, time.Time{}),
		datapoint.New(dfComplexUsed, dimensions, datapoint.NewIntValue(int64(fs.Used)), datapoint.Gauge, time.Time{}),
		datapoint.New(dfComplexReserved, dimensions, datapoint.NewIntValue(int64(fs.Reserved)), datapoint.Gauge, time.Time{}),
	}
}

func (fs *fsStat) makePercentBytes(dimensions map[string]string) []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		datapoint.New(percentBytesFree, dimensions, datapoint.NewFloatValue(100-fs.UsedPercent), datapoint.Gauge, time.Time{}),
		datapoint.New(percentBytesUsed, dimensions, datapoint.NewFloatValue(fs.UsedPercent), datapoint.Gauge, time.Time{}),
		datapoint.New(percentBytesReserved, dimensions, datapoint.NewFloatValue(fs.BytesReservedPercent), datapoint.Gauge, time.Time{}),
	}
}
