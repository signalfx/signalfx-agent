// +build windows

package filesystems

import (
	"fmt"
	"syscall"
	"testing"
	"unsafe"

	gopsutil "github.com/shirou/gopsutil/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

const uninitialized = 999
const closed = 998
const volumeA = "\\\\?\\Volume{11111111-0000-0000-0000-010000000000}\\"
const volumeC = "\\\\?\\Volume{22222222-0000-0000-0000-010000000000}\\"
const volumeD = "\\\\?\\Volume{33333333-0000-0000-0000-010000000000}\\"
const compressFlag = uint32(16)     // 0x00000010
const readOnlyFlag = uint32(524288) // 0x00080000
//type handle uintptr

type volumeMock struct {
	name string
	paths []string
	driveType uint32
	fsType string
	fsFlags uint32
}

type volumesMock struct {
	volumes   []volumeMock
}

func newVolumesMock() *volumesMock {
	volumes := make([]volumeMock, 0)
	volumes = append(volumes,
		volumeMock{name: volumeA, paths: []string{"A:\\"}, driveType: windows.DRIVE_REMOVABLE, fsType: "FAT16", fsFlags: compressFlag},
		volumeMock{name: volumeC, paths: []string{"C:\\"}, driveType: windows.DRIVE_FIXED, fsType: "NTFS", fsFlags: compressFlag},
		volumeMock{name: volumeC, paths: []string{"C:\\testHD\\"}, driveType: windows.DRIVE_FIXED, fsType: "NTFS", fsFlags: compressFlag},
		volumeMock{name: volumeD, paths: []string{"D:\\"}, driveType: windows.DRIVE_FIXED, fsType: "NTFS", fsFlags: compressFlag | readOnlyFlag},
	)

	u := uninitialized
	return &volumesMock{volumes: volumes}
}

func TestGetPartitionsWin_GetsAllPartitions(t *testing.T) {
	volumes := newVolumesMock()
	stats, err := getPartitionsWin(
		volumes.getDriveTypeMock,
		volumes.findFirstVolumeMock,
		volumes.findNextVolumeMock,
		volumes.findVolumeCloseMock,
		volumes.getVolumePathsMock,
		volumes.getVolumeInformationMock)
	fmt.Printf("PARTITION_STATS: %v\nERROR: %v\n", stats, err)
}

func (v *volumesMock) getDriveTypeMock(rootPath *uint16) (driveType uint32) {
	path := windows.UTF16PtrToString(rootPath)
	for _, info := range v.volumes {
		for _, p := range info.paths {
			if path == p {
				return info.driveType
			}
		}
	}
	return windows.DRIVE_UNKNOWN
}

func (v *volumesMock) findFirstVolumeMock(volumeNamePtr *uint16) (windows.Handle, error) {
	volumeIndex := 0
	//fmt.Printf("HANDLE: %d, FIRST_VOLUME_NAME: %s\n", volumeIndex, v.volumes[volumeIndex].name)
	//fmt.Printf("VOLUMES: %v\n", v)
	volumeName, err := windows.UTF16FromString(v.volumes[volumeIndex].name)
	if err != nil {
		return windows.Handle(volumeIndex), err
	}

	start := uintptr(unsafe.Pointer(volumeNamePtr))
	size := unsafe.Sizeof(*volumeNamePtr)
	for i := range volumeName {
		*(*uint16)(unsafe.Pointer(start + size * uintptr(i))) = volumeName[i]
	}

	return windows.Handle(uintptr(unsafe.Pointer(&volumeIndex))), nil
}

func (v *volumesMock) findNextVolumeMock(volumeIndexHandle windows.Handle, volumeNamePtr *uint16) error {
	volumeIndex := *(*int)(unsafe.Pointer(volumeIndexHandle))
	//fmt.Printf("VOLUMES_INDEX_START: %v\n", volumeIndex)
	if volumeIndex == uninitialized {
		return fmt.Errorf("find next volume handle uninitialized")
	}

	nextVolumeIndex := volumeIndex + 1
	if nextVolumeIndex >= len(v.volumes) {
		return syscall.Errno(18) // windows.ERROR_NO_MORE_FILES
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
	//fmt.Printf("VOLUMES_INDEX_END: %v\n", *(*int)(unsafe.Pointer(volumeIndexHandle)))

	return err
}

func (v *volumesMock) findVolumeCloseMock(volumeIndexHandle windows.Handle) error {
	volumeIndex := *(*int)(unsafe.Pointer(volumeIndexHandle))
	if volumeIndex != uninitialized {
		*(*int)(unsafe.Pointer(volumeIndexHandle)) = closed
	}
	return nil
}

func (v *volumesMock) getVolumePathsMock(volNameBuf []uint16) ([]string, error) {
	volumeName := windows.UTF16ToString(volNameBuf)
	for _, volume := range v.volumes {
		if volume.name == volumeName {
			return volume.paths, nil
		}
	}
	return nil, fmt.Errorf("path not found for volume: %s", volumeName)
}

func (v *volumesMock) getVolumeInformationMock(rootPath string, fsFlags *uint32, fsNameBuf []uint16) (err error) {
	for _, volume := range v.volumes {
		for _, path := range volume.paths {
			if rootPath == path {
				*fsFlags = volume.fsFlags
				fsNameBuf, err = windows.UTF16FromString(volume.name)
				return err
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

func TestGetPartitions_ShouldInclude_gopsutil_PartitionStats(t *testing.T) {
	// Partition stats from gopsutil are for drive mounts only.
	want, err := gopsutil.Partitions(true)
	require.NoError(t, err)

	require.NotEmpty(t, want, "failed to find any partition stats using gopsutil")

	//logger := logrus.WithFields(logrus.Fields{"monitorType": monitorType})
	//monitor := Monitor{logger: logger}

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
