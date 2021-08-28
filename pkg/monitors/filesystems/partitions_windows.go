// +build windows

package filesystems

import (
	"fmt"
	"strings"
	"syscall"

	gopsutil "github.com/shirou/gopsutil/disk"
	"golang.org/x/sys/windows"
)

var _ winPartitions = &Monitor{}

type winPartitions interface {
	partitionsWrapper
	getStatsDrive(all bool) ([]gopsutil.PartitionStat, error)
	getStatsFolder() ([]gopsutil.PartitionStat, error)
	getDriveType(rootPathName *uint16) (driveType uint32)
	findFirstVolume(volumeName *uint16, bufferLength uint32) (handle windows.Handle, err error)
	findNextVolume(findVolume windows.Handle, volumeName *uint16, bufferLength uint32) (err error)
	findVolumeClose(findVolume windows.Handle) (err error)
	getVolumePathNamesForVolumeName(volumeName *uint16, volumePathNames *uint16, bufferLength uint32, returnLength *uint32) (err error)
	getVolumeInformation(rootPathName *uint16, volumeNameBuffer *uint16, volumeNameSize uint32, volumeNameSerialNumber *uint32, maximumComponentLength *uint32, fileSystemFlags *uint32, fileSystemNameBuffer *uint16, fileSystemNameSize uint32) (err error)
}

// partitions returns partition stats of drive and folder mounts.
func (m *Monitor) getStats(all bool) (stats []gopsutil.PartitionStat, err error) {
	if stats, err = m.getStatsDrive(all); err != nil {
		return stats, err
	}
	fmt.Printf("STATS_DRIVE: %v\n", stats)
	var statsFolder []gopsutil.PartitionStat
	statsFolder, err = m.getStatsFolder()
	fmt.Printf("STATS_FOLDER: %v\n", statsFolder)
	return append(stats, statsFolder...), err
}

func (m *Monitor) getStatsDrive(all bool) ([]gopsutil.PartitionStat, error) {
	return gopsutil.Partitions(all)
}

// getStatsFolder returns partition stats of folder mounts.
// Similar to https://github.com/shirou/gopsutil/blob/7e4dab436b94d671021647dc5dc12c94f490e46e/disk/disk_windows.go#L71
func (m *Monitor) getStatsFolder() ([]gopsutil.PartitionStat, error) {
	statsFolders := make([]gopsutil.PartitionStat, 0)
	bufLen := uint32(syscall.MAX_PATH + 1)
	volNameBuf := make([]uint16, bufLen)

	handle, err := m.findFirstVolume(&volNameBuf[0], bufLen)
	if err != nil {
		return statsFolders, fmt.Errorf("failed to find first volume: %v", err)
	}
	defer m.findVolumeClose(handle)

	var volPathNames []uint16
	if volPathNames, err = m.getVolumePathNames(volNameBuf, bufLen); err != nil {
		return statsFolders, fmt.Errorf("failed to find paths for first volume %s: %v", windows.UTF16ToString(volNameBuf), err)
	}

	var statsFolder []gopsutil.PartitionStat
	statsFolder, err = m.newStats(volPathNames)
	if err != nil {
		return statsFolders, fmt.Errorf("failed to find partition stats for first volume %s: %v", windows.UTF16ToString(volNameBuf), err)
	}
	statsFolders = append(statsFolders, statsFolder...)

	for {
		volNameBuf = make([]uint16, bufLen)
		if err = m.findNextVolume(handle, &volNameBuf[0], bufLen); err != nil {
			if err.(syscall.Errno) == syscall.ERROR_NO_MORE_FILES {
				break
			}
			m.logger.WithError(err).Error("failed to find next volume")
			continue
		}

		driveType := m.getDriveType(&volNameBuf[0])
		fmt.Printf("VOLUME: %s, DRIVE_TYPE: %d\n", windows.UTF16ToString(volNameBuf), driveType)
		//if driveType != windows.DRIVE_NO_ROOT_DIR {
		//	continue
		//}

		volPathNames, err = m.getVolumePathNames(volNameBuf, bufLen)
		if err != nil {
			m.logger.WithError(err).Errorf("failed to find paths for volume %s", windows.UTF16ToString(volNameBuf))
			continue
		}
		fmt.Printf("VOLUME_PATH_NAMES: %s\n", windows.UTF16ToString(volPathNames))

		statsFolder, err = m.newStats(volPathNames)
		if err != nil {
			m.logger.WithError(err).Errorf("failed to find partition stats for volume %s", windows.UTF16ToString(volNameBuf))
			continue
		}
		statsFolders = append(statsFolders, statsFolder...)
	}

	return statsFolders, nil
}

func (m *Monitor) getDriveType(rootPathName *uint16) (driveType uint32) {
	return windows.GetDriveType(rootPathName)
}

func (m *Monitor) findFirstVolume(volumeName *uint16, bufferLength uint32) (handle windows.Handle, err error) {
	return windows.FindFirstVolume(volumeName, bufferLength)
}

func (m *Monitor) findNextVolume(findVolume windows.Handle, volumeName *uint16, bufferLength uint32) (err error) {
	return windows.FindNextVolume(findVolume, volumeName, bufferLength)
}

func (m *Monitor) findVolumeClose(findVolume windows.Handle) (err error) {
	return windows.FindVolumeClose(findVolume)
}

func (m *Monitor) getVolumePathNamesForVolumeName(volumeName *uint16, volumePathNames *uint16, bufferLength uint32, returnLength *uint32) (err error) {
	return windows.GetVolumePathNamesForVolumeName(volumeName, volumePathNames, bufferLength, returnLength)
}

func (m *Monitor) getVolumeInformation(rootPathName *uint16, volumeNameBuffer *uint16, volumeNameSize uint32, volumeNameSerialNumber *uint32, maximumComponentLength *uint32, fileSystemFlags *uint32, fileSystemNameBuffer *uint16, fileSystemNameSize uint32) (err error) {
	return windows.GetVolumeInformation(rootPathName, volumeNameBuffer, volumeNameSize, volumeNameSerialNumber, maximumComponentLength, fileSystemFlags, fileSystemNameBuffer, fileSystemNameSize)
}

// volumePaths returns the path for the given volume.
func (m *Monitor) getVolumePathNames(volNameBuf []uint16, bufLen uint32) ([]uint16, error) {
	volPathsBuf, returnLen := make([]uint16, bufLen), uint32(0)
	if err := m.getVolumePathNamesForVolumeName(&volNameBuf[0], &volPathsBuf[0], bufLen, &returnLen); err != nil {
		return nil, err
	}
	return volPathsBuf, nil
}

// newStats returns partition stats for the given volume path names (e.g. C:).
// Similar to https://github.com/shirou/gopsutil/blob/master/disk/disk_windows.go#L72
func (m *Monitor) newStats(volPathNames []uint16) ([]gopsutil.PartitionStat, error) {
	stats := make([]gopsutil.PartitionStat, 0)
	pathNames := strings.Split(strings.TrimRight(windows.UTF16ToString(volPathNames), "\x00"), "\x00")

	for _, pathName := range pathNames {
		fmt.Printf("PathName: %s\n", pathName)
		lpVolumeNameBuffer := make([]uint16, 256)
		lpVolumeSerialNumber := uint32(0)
		lpMaximumComponentLength := uint32(0)
		lpFileSystemFlags := uint32(0)
		lpFileSystemNameBuffer := make([]uint16, 256)
		path, _ := windows.UTF16PtrFromString(pathName + "\\")

		if err := m.getVolumeInformation(
			path,
			&lpVolumeNameBuffer[0],
			uint32(len(lpVolumeNameBuffer)),
			&lpVolumeSerialNumber,
			&lpMaximumComponentLength,
			&lpFileSystemFlags,
			&lpFileSystemNameBuffer[0],
			uint32(len(lpFileSystemNameBuffer)),
		); err != nil {
			return stats, err
		}

		opts := "rw"
		if int64(lpFileSystemFlags)&gopsutil.FileReadOnlyVolume != 0 {
			opts = "ro"
		}
		if int64(lpFileSystemFlags)&gopsutil.FileFileCompression != 0 {
			opts += ".compress"
		}

		stats = append(stats, gopsutil.PartitionStat{
			Device:     pathName,
			Mountpoint: pathName,
			Fstype:     windows.UTF16PtrToString(&lpFileSystemNameBuffer[0]),
			Opts:       opts,
		})
	}

	return stats, nil
}
