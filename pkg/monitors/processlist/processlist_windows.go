// +build windows

package processlist

import (
	"fmt"
	"time"

	"github.com/StackExchange/wmi"
	"github.com/shirou/gopsutil/mem"
	"golang.org/x/sys/windows"
)

const (
	processQueryLimitedInformation = 0x00001000
)

// Win32Process is a WMI struct used for WMI calls
// https://docs.microsoft.com/en-us/windows/desktop/CIMWin32Prov/win32-process
type Win32Process struct {
	Name           string
	ExecutablePath *string
	CommandLine    *string
	CreationDate   time.Time
	Priority       uint32
	ProcessID      uint32
	Status         *string
	ExecutionState *uint16
	KernelModeTime uint64
	PageFileUsage  uint32
	UserModeTime   uint64
	WorkingSetSize uint64
	VirtualSize    uint64
}

// PerfProcProcess is a performance process struct used for wmi calls
// https://msdn.microsoft.com/en-us/library/aa394323(v=vs.85).aspx
type PerfProcProcess struct {
	IDProcess            uint32
	PercentProcessorTime uint64
}

type osCache struct {
}

func initOSCache() *osCache {
	return &osCache{}
}

// getAllProcesses retrieves all processes.  It is set as a package variable so we can mock it during testing
var getAllProcesses = func() (ps []Win32Process, err error) {
	err = wmi.Query("select Name, ExecutablePath, CommandLine, CreationDate, Priority, ProcessID, Status, ExecutionState, KernelModeTime, PageFileUsage, UserModeTime, WorkingSetSize, VirtualSize from Win32_Process", &ps)
	return ps, err
}

// getUsername - retrieves a username from an open process handle it is set as a package variable so we can mock it during testing
var getUsername = func(id uint32) (username string, err error) {
	// open the process handle and collect any information that requires it
	var h windows.Handle
	defer func() { _ = windows.CloseHandle(h) }()
	if h, err = windows.OpenProcess(processQueryLimitedInformation, false, id); err != nil {
		err = fmt.Errorf("unable to open process handle. %v", err)
		return username, err
	}

	// the windows api docs suggest that windows.TOKEN_READ is a super set of windows.TOKEN_QUERY,
	// but in practice windows.TOKEN_READ seems to be less permissive for the admin user
	var token windows.Token
	defer token.Close()
	err = windows.OpenProcessToken(h, windows.TOKEN_QUERY, &token)
	if err != nil {
		err = fmt.Errorf("unable to retrieve process token. %v", err)
		return username, err
	}

	// extract the user from the process token
	user, err := token.GetTokenUser()
	if err != nil {
		err = fmt.Errorf("unable to get token user. %v", err)
		return username, err
	}

	// extract the username and domain from the user
	userid, domain, _, err := user.User.Sid.LookupAccount("")
	if err != nil {
		err = fmt.Errorf("unable to look up user account from Sid %v", err)
	}
	username = fmt.Sprintf("%s\\%s", domain, userid)

	return username, err
}

// ProcessList takes a snapshot of running processes
func ProcessList(conf *Config, cache *osCache) ([]*TopProcess, error) {
	var procs []*TopProcess

	// Get all processes
	ps, err := getAllProcesses()
	if err != nil {
		return nil, err
	}

	// iterate over each process and build an entry for the process list
	for _, p := range ps {
		username, err := getUsername(p.ProcessID)
		if err != nil {
			logger.Debugf("Unable to collect username for process %v. %v", p, err)
		}

		totalTime := time.Duration(float64(p.UserModeTime+p.KernelModeTime) * 100) // 100 ns units

		// Memory Percent
		var memPercent float64
		if systemMemory, err := mem.VirtualMemory(); err == nil {
			memPercent = 100 * float64(p.WorkingSetSize) / float64(systemMemory.Total)
		} else {
			logger.WithError(err).Error("Unable to collect system memory total")
		}

		// some windows processes do not have an executable path, but they do have a name
		command := *p.ExecutablePath
		if command == "" {
			command = p.Name
		}

		//example process "3":["root",20,"0",0,0,0,"S",0.0,0.0,"01:28.31","[ksoftirqd/0]"]
		procs = append(procs, &TopProcess{
			ProcessID:           int(p.ProcessID),
			CreatedTime:         p.CreationDate,
			Username:            username,
			Priority:            int(p.Priority),
			Nice:                nil, // nice value is not available on windows
			VirtualMemoryBytes:  p.VirtualSize,
			WorkingSetSizeBytes: p.WorkingSetSize,
			SharedMemBytes:      0,
			Status:              *p.Status,
			MemPercent:          memPercent,
			TotalCPUTime:        totalTime,
			Command:             command,
		})
	}
	return procs, nil
}
