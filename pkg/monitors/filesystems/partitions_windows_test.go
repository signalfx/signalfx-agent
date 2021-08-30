// +build windows

package filesystems

import (
	"fmt"
	"testing"

	gopsutil "github.com/shirou/gopsutil/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

const uninitialized uint = uint(999)
const closed uint = uint(998)
const volumeA = "\\\\?\\Volume{11111111-0000-0000-0000-010000000000}\\"
const volumeC = "\\\\?\\Volume{22222222-0000-0000-0000-010000000000}\\"
const volumeD = "\\\\?\\Volume{33333333-0000-0000-0000-010000000000}\\"
const compressFlag = uint32(16)     // 0x00000010
const readOnlyFlag = uint32(524288) // 0x00080000
//type handle uintptr

type volMock struct {
	name string
	paths []string
	driveType uint32
	fsType string
	fsFlags uint32
}

type volsMock struct {
	handle windows.Handle
	vols   []volMock
}

func newVolsMock() *volsMock {

	vols := make([]volMock, 0)
	vols = append(vols,
		volMock{name: volumeA, paths: []string{"A:\\"}, driveType: windows.DRIVE_REMOVABLE, fsType: "FAT16", fsFlags: compressFlag},
		volMock{name: volumeC, paths: []string{"C:\\"}, driveType: windows.DRIVE_FIXED, fsType: "NTFS", fsFlags: compressFlag},
		volMock{name: volumeC, paths: []string{"C:\\testHD\\"}, driveType: windows.DRIVE_FIXED, fsType: "NTFS", fsFlags: compressFlag},
		volMock{name: volumeD, paths: []string{"D:\\"}, driveType: windows.DRIVE_FIXED, fsType: "NTFS", fsFlags: compressFlag | readOnlyFlag},
	)

	return &volsMock{handle: windows.Handle(uninitialized), vols: vols}
}

func TestGetPartitionsWin_GetsAllPartitions(t *testing.T) {
	vols := newVolsMock()
	stats, err := getPartitionsWin(
		vols.getDriveTypeMock,
		vols.findFirstVolumeMock,
		vols.findNextVolumeMock,
		vols.findVolumeCloseMock,
		vols.getVolumePathsMock,
		vols.getVolumeInformationMock)
	fmt.Printf("PARTITION_STATS: %v\nERROR: %v\n", stats, err)
}

func (v *volsMock) getDriveTypeMock(rootPath *uint16) (driveType uint32) {
	path := windows.UTF16PtrToString(rootPath)
	for _, info := range v.vols {
		for _, p := range info.paths {
			if path == p {
				return info.driveType
			}
		}
	}
	return windows.DRIVE_UNKNOWN
}

func (v *volsMock) findFirstVolumeMock(volNamePtr *uint16) (windows.Handle, error) {
	findVol := uint(0)
	fmt.Printf("HANDLE: %d, FIRST_VOLUME_NAME: %s\n", findVol, v.vols[findVol].name)
	fmt.Printf("VOLUMES: %v\n", v)
	volName, err := windows.UTF16FromString(v.vols[findVol].name)
	if err != nil {
		return windows.Handle(findVol), err
	}
	size := unsafe.Sizeof(*volNamePtr)
	for i := range volName {
		*(*uint16)(unsafe.Add(unsafe.Pointer(volNamePtr), i*size)) = volName[i]
	}
	return windows.Handle(findVol), nil
}

func (v *volsMock) findNextVolumeMock(findVol windows.Handle, volNamePtr *uint16) error {
	if findVol == windows.Handle(uninitialized) {
		return fmt.Errorf("find next volume handle uninitialized")
	}

	findVol = findVol + 1
	volNamePtr, err := windows.UTF16PtrFromString(v.vols[findVol].name)
	return err
}

func (v *volsMock) findVolumeCloseMock(findVol windows.Handle) error {
	if findVol != windows.Handle(uninitialized) {
		findVol = windows.Handle(closed)
	}
	return nil
}

func (v *volsMock) getVolumePathsMock(volNameBuf []uint16) ([]string, error) {
	volName := windows.UTF16ToString(volNameBuf)
	for _, vol := range v.vols {
		if vol.name == volName {
			return vol.paths, nil
		}
	}
	return nil, fmt.Errorf("path not found for volume: %s", volName)
}

func (v *volsMock) getVolumeInformationMock(rootPath string, fsFlags *uint32, fsNameBuf []uint16) (err error) {
	for _, vol := range v.vols {
		for _, path := range vol.paths {
			if rootPath == path {
				*fsFlags = vol.fsFlags
				fsNameBuf, err = windows.UTF16FromString(vol.name)
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
