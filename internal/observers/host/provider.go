package host

import (
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
)

type hostInfoProvider interface {
	AllConnectionStats() ([]net.ConnectionStat, error)
	ProcessNameFromPID(pid int32) (string, error)
}

type defaultHostInfoProvider struct{}

// ConnectionStatProvider is what gives us our list of open sockets.
func (p *defaultHostInfoProvider) AllConnectionStats() ([]net.ConnectionStat, error) {
	return net.Connections("all")
}

// ProcessNameProvider is what looks up a process name from its pid.
func (p *defaultHostInfoProvider) ProcessNameFromPID(pid int32) (string, error) {
	proc, err := process.NewProcess(pid)
	if err != nil {
		return "", err
	}

	return proc.Name()
}
