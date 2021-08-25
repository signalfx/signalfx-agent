// +build windows

package filesystems

import (
	"strings"
	"syscall"

	gopsutil "github.com/shirou/gopsutil/disk"
	"golang.org/x/sys/windows"
)

// getPartitions returns partition stats of drive and folder mounts by adding the partition stats
// of folder mounts to partition stats of drive mounts returned gopsutil's Partitions(true).
func (m *Monitor) getPartitions(all bool) ([]gopsutil.PartitionStat, error) {
	// These partition stats from gopsutil are for drive mounts only.
	partStats, err := gopsutil.Partitions(all)
	if err != nil {
		return partStats, err
	}

	allMounts, folderMounts := m.getAllMounts(), make([]string, 0)
	for _, mnt := range allMounts {
		found := false
		for _, stats := range partStats {
			if mnt == stats.Mountpoint {
				found = true
				break
			}
		}
		if !found {
			folderMounts = append(folderMounts, mnt)
		}
	}

	var stats gopsutil.PartitionStat
	for _, mnt := range folderMounts {
		stats, err = newPartitionStats(mnt)
		if err != nil {
			m.logger.WithError(err).Errorf("failed to find partition stats for mount %s", mnt)
			continue
		}
		// Adding partition stats of folder mounts.
		partStats = append(partStats, stats)
	}

	return partStats, err
}

// getAllMounts gets the mount points for drive (C: etc.) and folder mounts.
// Similar to https://github.com/shirou/gopsutil/blob/7e4dab436b94d671021647dc5dc12c94f490e46e/disk/disk_windows.go#L71
func (m *Monitor) getAllMounts() []string {
	mounts := make([]string, 0)
	bufLen := uint32(syscall.MAX_PATH + 1)
	// Volume name buffer
	volNameBuf := make([]uint16, bufLen)

	handle, err := windows.FindFirstVolume(&volNameBuf[0], bufLen)
	if err != nil {
		m.logger.WithError(err).Errorf("failed to find first volume")
		return mounts
	}

	var volMounts []string
	if volMounts, err = volumeMounts(volNameBuf, bufLen); err != nil {
		m.logger.WithError(err).Errorf("failed to find mounts for first volume %s", windows.UTF16ToString(volNameBuf))
		return mounts
	}
	mounts = append(mounts, volMounts...)

	for {
		volNameBuf = make([]uint16, bufLen)
		err = windows.FindNextVolume(handle, &volNameBuf[0], bufLen)
		if err != nil {
			if err.(syscall.Errno) == syscall.ERROR_NO_MORE_FILES {
				break
			}
			m.logger.WithError(err).Error("failed to find next volume")
			continue
		}

		if volMounts, err = volumeMounts(volNameBuf, bufLen); err != nil {
			m.logger.WithError(err).Errorf("failed to find mounts for volume %s", windows.UTF16ToString(volNameBuf))
			continue
		}

		mounts = append(mounts, volMounts...)
	}

	_ = windows.FindVolumeClose(handle)

	for i := range mounts {
		mounts[i] = strings.TrimRight(mounts[i], "\\")
	}

	return mounts
}

// volumeMounts returns the mount points for the given volume.
func volumeMounts(volNameBuf []uint16, bufLen uint32) ([]string, error) {
	volPathsBuf, returnLen := make([]uint16, bufLen), uint32(0)
	if err := windows.GetVolumePathNamesForVolumeName(&volNameBuf[0], &volPathsBuf[0], bufLen, &returnLen); err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimRight(windows.UTF16ToString(volPathsBuf), "\x00"), "\x00"), nil
}

// newPartitionStats returns partition stats for the given mount.
// Similar to https://github.com/shirou/gopsutil/blob/master/disk/disk_windows.go#L72
func newPartitionStats(mount string) (gopsutil.PartitionStat, error) {
	lpVolumeNameBuffer := make([]uint16, 256)
	lpVolumeSerialNumber := uint32(0)
	lpMaximumComponentLength := uint32(0)
	lpFileSystemFlags := uint32(0)
	lpFileSystemNameBuffer := make([]uint16, 256)
	volPath, _ := windows.UTF16PtrFromString(mount + "\\")

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
		return gopsutil.PartitionStat{}, err
	}

	opts := "rw"
	if int64(lpFileSystemFlags)&gopsutil.FileReadOnlyVolume != 0 {
		opts = "ro"
	}
	if int64(lpFileSystemFlags)&gopsutil.FileFileCompression != 0 {
		opts += ".compress"
	}

	return gopsutil.PartitionStat{
		Device:     mount,
		Mountpoint: mount,
		Fstype:     windows.UTF16PtrToString(&lpFileSystemNameBuffer[0]),
		Opts:       opts,
	}, nil
}
