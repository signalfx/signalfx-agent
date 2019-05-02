// +build windows

package processlist

import (
	"bytes"
	"fmt"
	"math"

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

// toTime returns the given seconds as a formatted string "min:sec.dec"
func toTime(secs float64) string {
	minutes := int(secs) / 60
	seconds := math.Mod(secs, 60.0)
	dec := math.Mod(seconds, 1.0) * 100
	return fmt.Sprintf("%02d:%02.f.%02.f", minutes, seconds, dec)
}

// getCPUPercentages is set as a package variable so we can mock it during testing
var getCPUPercentages = func() (cpuPercents map[uint32]uint64, err error) {
	// Get all process cpu percentages
	var processes []PerfProcProcess
	err = wmi.Query("select IDProcess, PercentProcessorTime from Win32_PerfFormattedData_PerfProc_Process where Name != '_Total'", &processes)
	cpuPercents = make(map[uint32]uint64, len(processes))
	for _, p := range processes {
		cpuPercents[p.IDProcess] = p.PercentProcessorTime
	}
	return cpuPercents, err
}

// getAllProcesses retrieves all processes.  It is set as a package variable so we can mock it during testing
var getAllProcesses = func() (ps []Win32Process, err error) {
	err = wmi.Query("select Name, ExecutablePath, CommandLine, Priority, ProcessID, Status, ExecutionState, KernelModeTime, PageFileUsage, UserModeTime, WorkingSetSize, VirtualSize from Win32_Process", &ps)
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

// ProcessList takes a snapshot of running processes and returns a byte buffer
func ProcessList() (*bytes.Buffer, error) {
	processes := &bytes.Buffer{}
	processes.WriteString("{")
	defer processes.WriteString("}") // always close the associative array

	// Get all processes
	ps, err := getAllProcesses()
	if err != nil {
		return processes, err
	}

	// Get cpu percentages for all processes
	cpuPercentages, err := getCPUPercentages()
	if err != nil {
		logger.Debugf("No per process cpu percentages returned %v", err)
	}

	// index position to stop appending commas to the list of processes
	stop := len(ps) - 1

	// iterate over each process and build an entry for the process list
	for index, p := range ps {
		username, err := getUsername(p.ProcessID)
		if err != nil {
			logger.Debugf("Unable to collect username for process %v. %v", p, err)
		}

		// CPU Times
		var cpuPercent float64
		totalTime := float64(p.UserModeTime+p.KernelModeTime) / 10000000 // 100 ns units to seconds
		if percent, inMap := cpuPercentages[p.ProcessID]; inMap {
			cpuPercent = float64(percent)
		} else {
			logger.Debugf("Unable to find cpu percentage for pid %d.", p.ProcessID)
		}

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
		fmt.Fprintf(processes, "\"%d\":[\"%s\",%d,\"%s\",%d,%d,%d,\"%s\",%.2f,%.2f,\"%s\",\"%s\"]",
			p.ProcessID,           // pid
			username,              // username
			p.Priority,            // priority
			"",                    // nice value is not available on windows
			p.PageFileUsage/1024,  // virtual memory size in kb
			p.WorkingSetSize/1024, // resident memory size in kb
			0/1024,                // shared memory
			*p.Status,             // status
			cpuPercent,            // % cpu, float
			memPercent,            // % mem, float
			toTime(totalTime),     // cpu time
			command,               // command/executable
		)

		// append a comma as long as it isn't the last entry in the associative array
		if index != stop {
			processes.WriteString(",")
		}
	}
	return processes, nil
}
