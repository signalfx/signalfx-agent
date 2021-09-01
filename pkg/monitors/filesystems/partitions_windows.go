// +build windows

package filesystems

import (
	"strings"
	"syscall"
	"unicode/utf16"

	"github.com/pkg/errors"
	gopsutil "github.com/shirou/gopsutil/disk"
	"golang.org/x/sys/windows"
)

const volumeNameBufferLength = uint32(windows.MAX_PATH + 1)
const volumePathBufferLength = volumeNameBufferLength

func getPartitions(all bool) ([]gopsutil.PartitionStat, error) {
	return getPartitionsWin(getDriveType, findFirstVolume, findNextVolume, findVolumeClose, getVolumePaths, getVolumeInformation)
}

// getPartitions returns partition stats.
// Similar to https://github.com/shirou/gopsutil/blob/7e4dab436b94d671021647dc5dc12c94f490e46e/disk/disk_windows.go#L71
func getPartitionsWin(
	getDriveType func(rootPath string) (driveType uint32),
	findFirstVolume func(volName *uint16) (findVol windows.Handle, err error),
	findNextVolume func(findVol windows.Handle, volName *uint16) (err error),
	findVolumeClose func(findVol windows.Handle) (err error),
	getVolumePaths func(volNameBuf []uint16) ([]string, error),
	getVolumeInformation func(rootPath string, fsFlags *uint32, fsNameBuf []uint16) (err error),
) ([]gopsutil.PartitionStat, error) {

	stats := make([]gopsutil.PartitionStat, 0)
	volNameBuf := make([]uint16, volumeNameBufferLength)

	handle, err := findFirstVolume(&volNameBuf[0])
	if err != nil {
		return stats, errors.WithMessagef(err, "cannot find first volume")
	}
	defer findVolumeClose(handle)

	var volPaths []string
	if volPaths, err = getVolumePaths(volNameBuf); err != nil {
		return stats, errors.WithMessagef(err, "cannot find paths for first volume %s", windows.UTF16ToString(volNameBuf))
	}

	var partitionStats []gopsutil.PartitionStat
	partitionStats, err = getPartitionStats(getDriveType(volPaths[0]), volPaths, getVolumeInformation)
	if err != nil {
		return stats, errors.WithMessagef(err, "cannot find partition stats for first volume %s", windows.UTF16ToString(volNameBuf))
	}
	stats = append(stats, partitionStats...)

	var lastError error
	for {
		volNameBuf = make([]uint16, volumeNameBufferLength)
		if err = findNextVolume(handle, &volNameBuf[0]); err != nil {
			if errno, ok := err.(syscall.Errno); ok && errno == windows.ERROR_NO_MORE_FILES {
				break
			}
			lastError = errors.WithMessagef(err, "last error of find next volume error(s)")
			continue
		}

		volPaths, err = getVolumePaths(volNameBuf)
		if err != nil {
			lastError = errors.WithMessagef(err, "last error of find paths error(s) for volume %s", windows.UTF16ToString(volNameBuf))
			continue
		}

		partitionStats, err = getPartitionStats(getDriveType(volPaths[0]), volPaths, getVolumeInformation)
		if err != nil {
			lastError = errors.WithMessagef(err, "last error of find partition stats error(s) for volume %s", windows.UTF16ToString(volNameBuf))
			continue
		}
		stats = append(stats, partitionStats...)
	}

	return stats, lastError
}

func getDriveType(rootPath string) (driveType uint32) {
	rootPathPtr, _ := windows.UTF16PtrFromString(rootPath)
	return windows.GetDriveType(rootPathPtr)
}

func findFirstVolume(volName *uint16) (findVol windows.Handle, err error) {
	return windows.FindFirstVolume(volName, volumeNameBufferLength)
}

func findNextVolume(findVol windows.Handle, volName *uint16) (err error) {
	return windows.FindNextVolume(findVol, volName, volumeNameBufferLength)
}

func findVolumeClose(findVol windows.Handle) (err error) {
	return windows.FindVolumeClose(findVol)
}

// getVolumePaths returns the path for the given volume.
func getVolumePaths(volNameBuf []uint16) ([]string, error) {
	volPathsBuf := make([]uint16, volumePathBufferLength)
	returnLen := uint32(0)
	if err := windows.GetVolumePathNamesForVolumeName(&volNameBuf[0], &volPathsBuf[0], volumePathBufferLength, &returnLen); err != nil {
		return nil, err
	}

	return findStrings(volPathsBuf, int(returnLen)), nil
}

func findStrings(ss16 []uint16, size int) []string {
	if len(ss16) < size {
		return nil
	}
	ss := make([]string, 0)
	from := 0
	for to := 0; to < size; to++ {
		if ss16[to] == 0 {
			if from < to && ss16[from] != 0 {
				ss = append(ss, string(utf16.Decode(ss16[from:to])))
			}
			from = to + 1
		}
	}
	return ss
}

func getVolumeInformation(rootPath string, fsFlags *uint32, fsNameBuf []uint16) error {
	volNameBuf := make([]uint16, 256)
	volSerialNum := uint32(0)
	maxComponentLen := uint32(0)
	rootPathPtr, _ := windows.UTF16PtrFromString(rootPath)

	return windows.GetVolumeInformation(
		rootPathPtr,
		&volNameBuf[0],
		uint32(len(volNameBuf)),
		&volSerialNum,
		&maxComponentLen,
		fsFlags,
		&fsNameBuf[0],
		uint32(len(fsNameBuf)))
}

// getPartitionStats returns partition stats for the given volume path.
// Similar to https://github.com/shirou/gopsutil/blob/master/disk/disk_windows.go#L72
func getPartitionStats(
	driveType uint32,
	volPaths []string,
	getVolumeInformation func(rootPath string, fsFlags *uint32, fsNameBuf []uint16) (err error),
) ([]gopsutil.PartitionStat, error) {

	stats := make([]gopsutil.PartitionStat, 0)

	var lastError error
	for _, volPath := range volPaths {
		if driveType == windows.DRIVE_REMOVABLE || driveType == windows.DRIVE_FIXED || driveType == windows.DRIVE_REMOTE || driveType == windows.DRIVE_CDROM {
			fsFlags, fsNameBuf := uint32(0), make([]uint16, 256)

			if err := getVolumeInformation(volPath, &fsFlags, fsNameBuf); err != nil {
				lastError = errors.WithMessagef(err, "last error of error(s) in getting volume informaton")
				if driveType == windows.DRIVE_CDROM || driveType == windows.DRIVE_REMOVABLE {
					continue //device is not ready will happen if there is no disk in the drive
				}
				return stats, lastError
			}

			opts := "rw"
			if int64(fsFlags)&gopsutil.FileReadOnlyVolume != 0 {
				opts = "ro"
			}
			if int64(fsFlags)&gopsutil.FileFileCompression != 0 {
				opts += ".compress"
			}

			p := strings.TrimRight(volPath, "\\")
			stats = append(stats, gopsutil.PartitionStat{
				Device:     p,
				Mountpoint: p,
				Fstype:     windows.UTF16PtrToString(&fsNameBuf[0]),
				Opts:       opts,
			})
		}
	}

	return stats, lastError
}
