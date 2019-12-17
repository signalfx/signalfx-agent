// +build !windows

package cpu

import (
	"github.com/shirou/gopsutil/cpu"
)

// setting cpu.Times to a package variable for testing purposes
var times = cpu.Times
