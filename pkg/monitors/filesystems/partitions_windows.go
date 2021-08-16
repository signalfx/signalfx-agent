// +build windows

package filesystems

import (
	"strings"
	"syscall"

	gopsutil "github.com/shirou/gopsutil/disk"
	"golang.org/x/sys/windows"
)

func (m *Monitor) getPartitions(all bool) ([]gopsutil.PartitionStat, error) {
	drivePartitionStats, err := gopsutil.Partitions(all)
	if err != nil {
		return drivePartitionStats, err
	}

	allMountPoints := m.getAllMountPoints()
	folderMountPoints := make([]string, 0)

	for _, mountPoint := range allMountPoints {
		found := false
		for _, drivePartitionStat := range drivePartitionStats {
			if mountPoint == drivePartitionStat.Mountpoint {
				found = true
				break
			}
		}
		if !found {
			folderMountPoints = append(folderMountPoints, mountPoint)
		}
	}

	drivePartitionStats = append(drivePartitionStats, m.newPartitionStats(folderMountPoints)...)

	return drivePartitionStats, err

}

// getAllMountPoints gets mount points for all volumes (letter drive (C: etc.) and folder mounted volumes).
func (m *Monitor) getAllMountPoints() []string {
	mountPoints := make([]string, 0)
	bufLen := uint32(syscall.MAX_PATH + 1)
	volNameBuf := make([]uint16, bufLen)

	handle, err := windows.FindFirstVolume(&volNameBuf[0], bufLen)
	if err != nil {
		m.logger.WithError(err).Errorf("failed to find mount points")
		return mountPoints
	}

	for {
		volNameBuf = make([]uint16, bufLen)
		err = windows.FindNextVolume(handle, &volNameBuf[0], bufLen)
		if err != nil {
			if err.(syscall.Errno) == syscall.ERROR_NO_MORE_FILES {
				break
			}
			m.logger.WithError(err).Errorf("failed to find mount points for volume %s", windows.UTF16ToString(volNameBuf))
			continue
		}

		volPathsBuf := make([]uint16, bufLen)
		returnLen := uint32(0)
		if err = windows.GetVolumePathNamesForVolumeName(&volNameBuf[0], &volPathsBuf[0], bufLen, &returnLen); err != nil {
			m.logger.WithError(err).Errorf("failed to find mount points for volume %s", windows.UTF16ToString(volNameBuf))
			continue
		}

		volPaths := strings.Split(strings.TrimRight(windows.UTF16ToString(volPathsBuf), "\x00"), "\x00")
		mountPoints = append(mountPoints, volPaths...)
	}

	for i := range mountPoints {
		mountPoints[i] = strings.TrimRight(mountPoints[i], "\\")
	}
	return mountPoints
}

func (m *Monitor) newPartitionStats(mountPoints []string) []gopsutil.PartitionStat {
	stats := make([]gopsutil.PartitionStat, 0)

	for _, mountPoint := range mountPoints {
		lpVolumeNameBuffer := make([]uint16, 256)
		lpVolumeSerialNumber := uint32(0)
		lpMaximumComponentLength := uint32(0)
		lpFileSystemFlags := uint32(0)
		lpFileSystemNameBuffer := make([]uint16, 256)
		volPath, _ := windows.UTF16PtrFromString(mountPoint + "\\")

		if err := windows.GetVolumeInformation(
			volPath,
			&lpVolumeNameBuffer[0],
			uint32(len(lpVolumeNameBuffer)),
			&lpVolumeSerialNumber,
			&lpMaximumComponentLength,
			&lpFileSystemFlags,
			&lpFileSystemNameBuffer[0],
			uint32(len(lpFileSystemNameBuffer)),
		); err != nil {
			m.logger.WithError(err).Errorf("failed to find volume information for mountpoint `%s`", mountPoint)
			continue
		}

		opts := "rw"
		if int64(lpFileSystemFlags)&gopsutil.FileReadOnlyVolume != 0 {
			opts = "ro"
		}
		if int64(lpFileSystemFlags)&gopsutil.FileFileCompression != 0 {
			opts += ".compress"
		}

		stats = append(stats, gopsutil.PartitionStat{
			Device:     mountPoint,
			Mountpoint: mountPoint,
			Fstype:     windows.UTF16PtrToString(&lpFileSystemNameBuffer[0]),
			Opts:       opts,
		})
	}

	return stats
}
