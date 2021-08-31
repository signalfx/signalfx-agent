// +build windows

package filesystems

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	gopsutil "github.com/shirou/gopsutil/disk"
	"golang.org/x/sys/windows"
)

const volumeNameBufferLength = uint32(syscall.MAX_PATH + 1)
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
		return stats, fmt.Errorf("failed to find first volume: %v", err)
	}
	defer findVolumeClose(handle)

	fmt.Printf("HANDLE_AFTER_findFirstVolume: %v\n", *(*int)(unsafe.Pointer(handle)))

	var volPaths []string
	if volPaths, err = getVolumePaths(volNameBuf); err != nil {
		return stats, fmt.Errorf("failed to find paths for first volume %s: %v", windows.UTF16ToString(volNameBuf), err)
	}
	fmt.Printf("HANDLE_AFTER_getVolumePaths: %v\n", *(*int)(unsafe.Pointer(handle)))
	var partitionStats []gopsutil.PartitionStat
	partitionStats, err = getPartitionStats(getDriveType(volPaths[0]), volPaths, getVolumeInformation)
	if err != nil {
		return stats, fmt.Errorf("failed to find partition stats for first volume %s: %v", windows.UTF16ToString(volNameBuf), err)
	}
	stats = append(stats, partitionStats...)

	fmt.Printf("HANDLE_AFTER_getPartitionStats: %v\n", *(*int)(unsafe.Pointer(handle)))
	var lastError error
	for {
		volNameBuf = make([]uint16, volumeNameBufferLength)
		if err = findNextVolume(handle, &volNameBuf[0]); err != nil {
			if err.(syscall.Errno) == windows.ERROR_NO_MORE_FILES {
				break
			}
			lastError = fmt.Errorf("last error: failed to find next volume: %v", err)
			continue
		}
		fmt.Printf("HANDLE_AFTER_NEXT: %v\n", *(*int)(unsafe.Pointer(handle)))

		volPaths, err = getVolumePaths(volNameBuf)
		if err != nil {
			lastError = fmt.Errorf("last error: failed to find paths for volume %s: %v", windows.UTF16ToString(volNameBuf), err)
			continue
		}

		partitionStats, err = getPartitionStats(getDriveType(volPaths[0]), volPaths, getVolumeInformation)
		if err != nil {
			lastError = fmt.Errorf("last error: failed to find partition stats for volume %s: %v", windows.UTF16ToString(volNameBuf), err)
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

// volumePaths returns the path for the given volume.
func getVolumePaths(volNameBuf []uint16) ([]string, error) {
	volPathsBuf := make([]uint16, volumePathBufferLength)
	returnLen := uint32(0)
	if err := windows.GetVolumePathNamesForVolumeName(&volNameBuf[0], &volPathsBuf[0], volumePathBufferLength, &returnLen); err != nil {
		return nil, err
	}

	volPaths := make([]string, 0)
	for _, volPath := range strings.Split(strings.TrimRight(windows.UTF16ToString(volPathsBuf), "\x00"), "\x00") {
		volPaths = append(volPaths, volPath)
	}

	return volPaths, nil
}

func getVolumeInformation(rootPath string, fsFlags *uint32, fsNameBuf []uint16) (err error) {
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
		fsFlags := uint32(0)
		fsNameBuf := make([]uint16, 256)

		if err := getVolumeInformation(volPath, &fsFlags, fsNameBuf); err != nil {
			lastError = fmt.Errorf("last error: failed to get volume informaton: %v", err)
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

	return stats, lastError
}
