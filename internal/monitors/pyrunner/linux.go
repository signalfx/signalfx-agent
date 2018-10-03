// +build linux

package pyrunner

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
)

// The Linux specific process attribute that make the Python runner be in the
// same process group as the agent so they get shutdown together.
func procAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		// This is Linux-specific and will cause collectd to be killed by the OS if
		// the agent dies
		Pdeathsig: syscall.SIGTERM,
	}
}

func pythonBinaryExecutable() string {
	return filepath.Join(os.Getenv(constants.BundleDirEnvVar), "lib64/ld-linux-x86-64.so.2")
}

func pythonBinaryArgs(pkgName string) []string {
	return []string{
		filepath.Join(os.Getenv(constants.BundleDirEnvVar), "bin/python"),
		"-u",
		"-m",
		pkgName,
	}
}
