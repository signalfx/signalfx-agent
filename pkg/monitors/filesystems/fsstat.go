// +build !linux

package filesystems

import (
	"time"

	gopsutil "github.com/shirou/gopsutil/disk"
	"github.com/signalfx/golib/v3/datapoint"
)

type fsStat struct {
	*gopsutil.UsageStat
}

// newFsStat returns a file system usage. path is a filesystem path such as "/", not device file path like
// "/dev/vda1". If you want to use a return value of disk.Partitions, use "Mountpoint" not "Device".
func newFsStat(path string) (*fsStat, error) {
	disk, err := gopsutil.Usage(path)
	if err != nil {
		return nil, err
	}
	return &fsStat{disk}, nil
}

func (fs *fsStat) diskUtilizationDatapoint(dimensions map[string]string) *datapoint.Datapoint {
	return datapoint.New(diskUtilization, dimensions, datapoint.NewFloatValue(fs.UsedPercent), datapoint.Gauge, time.Time{})
}

func (fs *fsStat) makeInodeDatapoints(dimensions map[string]string) []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		datapoint.New(dfInodesFree, dimensions, datapoint.NewIntValue(int64(fs.InodesFree)), datapoint.Gauge, time.Time{}),
		datapoint.New(dfInodesUsed, dimensions, datapoint.NewIntValue(int64(fs.InodesUsed)), datapoint.Gauge, time.Time{}),
		// TODO: implement df_inodes.reserved
		datapoint.New(percentInodesFree, dimensions, datapoint.NewIntValue(int64(100-fs.InodesUsedPercent)), datapoint.Gauge, time.Time{}),
		datapoint.New(percentInodesUsed, dimensions, datapoint.NewIntValue(int64(fs.InodesUsedPercent)), datapoint.Gauge, time.Time{}),
		// TODO: implement percent_inodes.reserved
	}
}

func (fs *fsStat) makeDFComplex(dimensions map[string]string) []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		datapoint.New(dfComplexFree, dimensions, datapoint.NewIntValue(int64(fs.Free)), datapoint.Gauge, time.Time{}),
		datapoint.New(dfComplexUsed, dimensions, datapoint.NewIntValue(int64(fs.Used)), datapoint.Gauge, time.Time{}),
		// TODO: implement df_complex.reserved
	}
}

func (fs *fsStat) makePercentBytes(dimensions map[string]string) []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		datapoint.New(percentBytesFree, dimensions, datapoint.NewFloatValue(100-fs.UsedPercent), datapoint.Gauge, time.Time{}),
		datapoint.New(percentBytesUsed, dimensions, datapoint.NewFloatValue(fs.UsedPercent), datapoint.Gauge, time.Time{}),
		// TODO: implement percent_bytes.reserved
	}
}
