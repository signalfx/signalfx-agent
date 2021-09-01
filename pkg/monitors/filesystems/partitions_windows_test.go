// +build windows

package filesystems

import (
	"fmt"
	"testing"
	"unsafe"

	gopsutil "github.com/shirou/gopsutil/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

const uninitialized = 999
const closed = 998
const compressFlag = uint32(16)     // 0x00000010
const readOnlyFlag = uint32(524288) // 0x00080000
//type handle uintptr

type volumeMock struct {
	name string
	paths []string
	driveType uint32
	fsType string
	fsFlags uint32
	err error
}

type volumesMock struct {
	handle int
	volumes []volumeMock
}

var driveVolume = volumeMock{
	name: "\\\\?\\Volume{1e1e1111-0000-0000-0000-010000000000}\\",
	paths: []string{"C:\\"},
	driveType: windows.DRIVE_FIXED,
	fsType: "NTFS",
	fsFlags: compressFlag,
	err: nil}

var driveAndFolderVolume = volumeMock{
	name: "\\\\?\\Volume{0000cccc-0000-0000-0000-010000000000}\\",
	paths: []string{"D:\\", "C:\\mnt\\driveD\\"},
	driveType: windows.DRIVE_FIXED,
	fsType: "NTFS",
	fsFlags: compressFlag | readOnlyFlag,
	err: nil}

var removableDriveVolume = volumeMock{
	name: "\\\\?\\Volume{bbbbaaaa-0000-0000-0000-010000000000}\\",
	paths: []string{"A:\\"},
	driveType: windows.DRIVE_REMOVABLE,
	fsType: "FAT16",
	fsFlags: compressFlag,
	err: nil}

func TestGetPartitionsWin(t *testing.T) {
	tests := []struct {
		name string
		volumes *volumesMock
		want []gopsutil.PartitionStat
	}{
		{
			name: "all partition stats given no errors",
			volumes: func() *volumesMock {
				vols := append(make([]volumeMock, 0), driveVolume, driveAndFolderVolume, removableDriveVolume)
				return &volumesMock{handle: 0, volumes: vols}
			}(),
			want: []gopsutil.PartitionStat{
				{Device: "C:", Mountpoint: "C:", Fstype: "NTFS", Opts: "rw.compress"},
				{Device: "D:", Mountpoint: "D:", Fstype: "NTFS", Opts: "ro.compress"},
				{Device: "C:\\mnt\\driveD", Mountpoint: "C:\\mnt\\driveD", Fstype: "NTFS", Opts: "ro.compress"},
				{Device: "A:", Mountpoint: "A:", Fstype: "FAT16", Opts: "ro.compress"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stats, err := getPartitionsWin(
				test.volumes.getDriveTypeMock,
				test.volumes.findFirstVolumeMock,
				test.volumes.findNextVolumeMock,
				test.volumes.findVolumeCloseMock,
				test.volumes.getVolumePathsMock,
				test.volumes.getVolumeInformationMock)

			require.NoError(t, err)
			require.Equal(t, len(test.want), len(stats), "Number of partition stats not equal to expected")

			for i := 0; i < len(stats); i++ {
				assert.Equal(t, test.want[i], stats[i])
			}
		})
	}
}

func (v *volumesMock) getDriveTypeMock(rootPath string) (driveType uint32) {
	for _, volume := range v.volumes {
		for _, path := range volume.paths {
			if path == rootPath {
				return volume.driveType
			}
		}
	}
	return windows.DRIVE_UNKNOWN
}

func (v *volumesMock) findFirstVolumeMock(volumeNamePtr *uint16) (windows.Handle, error) {
	volumeName, err := windows.UTF16FromString(v.volumes[v.handle].name)
	if err != nil {
		return windows.Handle(unsafe.Pointer(&v.handle)), err
	}

	start := uintptr(unsafe.Pointer(volumeNamePtr))
	size := unsafe.Sizeof(*volumeNamePtr)
	for i := range volumeName {
		*(*uint16)(unsafe.Pointer(start + size * uintptr(i))) = volumeName[i]
	}

	return windows.Handle(unsafe.Pointer(&v.handle)), nil
}

func (v *volumesMock) findNextVolumeMock(volumeIndexHandle windows.Handle, volumeNamePtr *uint16) error {
	volumeIndex := *(*int)(unsafe.Pointer(volumeIndexHandle))
	if volumeIndex == uninitialized {
		return fmt.Errorf("find next volume handle uninitialized")
	}

	nextVolumeIndex := volumeIndex + 1
	if nextVolumeIndex >= len(v.volumes) {
		return windows.ERROR_NO_MORE_FILES
	}

	volumeName, err := windows.UTF16FromString(v.volumes[nextVolumeIndex].name)
	if err != nil {
		return err
	}

	start := uintptr(unsafe.Pointer(volumeNamePtr))
	size := unsafe.Sizeof(*volumeNamePtr)
	for i := range volumeName {
		*(*uint16)(unsafe.Pointer(start + size * uintptr(i))) = volumeName[i]
	}

	*(*int)(unsafe.Pointer(volumeIndexHandle)) = nextVolumeIndex

	return err
}

func (v *volumesMock) findVolumeCloseMock(volumeIndexHandle windows.Handle) error {
	volumeIndex := *(*int)(unsafe.Pointer(volumeIndexHandle))
	if volumeIndex != uninitialized {
		*(*int)(unsafe.Pointer(volumeIndexHandle)) = closed
	}
	return nil
}

func (v *volumesMock) getVolumePathsMock(volNameBuf []uint16) (volumePaths []string, err error) {
	volumeName := windows.UTF16ToString(volNameBuf)
	for _, volume := range v.volumes {
		if volume.name == volumeName {
			volumePaths = append(volumePaths, volume.paths...)
		}
	}
	if len(volumePaths) == 0 {
		err = fmt.Errorf("path not found for volume: %s", volumeName)
	}
	return volumePaths, err
}

func (v *volumesMock) getVolumeInformationMock(rootPath string, fsFlags *uint32, fsNameBuf []uint16) error {
	for _, volume := range v.volumes {
		for _, path := range volume.paths {
			if rootPath == path {
				*fsFlags = volume.fsFlags
				fsName, err := windows.UTF16FromString(volume.fsType)
				if err != nil {
					return err
				}
				for i := range fsName {
					fsNameBuf[i] = fsName[i]
				}
				return nil
			}
		}
	}
	return fmt.Errorf("cannot find volume information for volume path %s", rootPath)
}

//func TestGetAllMounts_ShouldInclude_gopsutil_Mounts(t *testing.T) {
//	logger := logrus.WithFields(logrus.Fields{"monitorType": monitorType})
//
//	// Drive and folder mounts.
//	got := (&Monitor{logger: logger}).getAllMounts()
//	require.NotEmpty(t, got, "failed to find any mount points")
//
//	// Mounts from gopsutil are for drives only.
//	want, err := getGopsutilMounts()
//	require.NoError(t, err)
//
//	require.NotEmpty(t, want, "failed to find any mount points using gopsutil")
//
//	// Asserting `got` getAllMounts() mounts superset of `want` gopsutil mounts.
//	assert.Subset(t, got, want)
//}

//func TestNewStats_SameAs_gopsutil_PartitionStats(t *testing.T) {
//	// Partition stats from gopsutil are for drive mounts only.
//	gopsutilStats, err := gopsutil.Partitions(true)
//	require.NoError(t, err)
//
//	require.NotEmpty(t, gopsutilStats, "failed to find any partition stats using gopsutil")
//
//	logger := logrus.WithFields(logrus.Fields{"monitorType": monitorType})
//	monitor := Monitor{logger: logger}
//
//	var got []gopsutil.PartitionStat
//	for _, want := range gopsutilStats {
//		volPathName, _ := windows.UTF16FromString(want.Mountpoint)
//		got, err = monitor.getStats(volPathName)
//		require.NoError(t, err)
//
//		// Asserting `got` getStats() stats equal `want` gopsutil stats.
//		assert.Equal(t, got[0], want)
//	}
//}

//VOLUME: \\?\Volume{692d8a75-0000-0000-0000-100000000000}\, PATHS: [C:\]
//VOLUME: \\?\Volume{bf5d138f-0000-0000-0000-010000000000}\, PATHS: [C:\mnt\testHD\]
//        \\?\Volume{bf5d138f-0000-0000-0000-010000000000}\
//VOLUME: \\?\Volume{bf5d0775-0000-0000-0000-010000000000}\, PATHS: [D:\]

func TestGetPartitions_ShouldInclude_gopsutil_PartitionStats(t *testing.T) {
	// Partition stats from gopsutil are for drive mounts only.
	want, err := gopsutil.Partitions(true)
	require.NoError(t, err)

	require.NotEmpty(t, want, "failed to find any partition stats using gopsutil")

	var got []gopsutil.PartitionStat
	// Partition stats for drive and folder mounts.
	got, err = getPartitions(true)
	require.NoError(t, err)

	require.NotEmpty(t, got, "failed to find any partition stats")

	// Asserting `got` getPartitions stats superset of `want` gopsutil stats.
	assert.Subset(t, got, want)
}

//func getGopsutilMounts() ([]string, error) {
//	partitionsStats, err := gopsutil.Partitions(true)
//	if err != nil {
//		return nil, err
//	}
//
//	mounts := make([]string, 0)
//	for _, stats := range partitionsStats {
//		mounts = append(mounts, stats.Mountpoint)
//	}
//
//	return mounts, nil
//}
