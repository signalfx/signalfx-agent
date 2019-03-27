// +build linux

package pyrunner

import (
	"os"
	"path/filepath"
	"runtime"
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

func defaultPythonBinaryExecutable() []string {
	loader := filepath.Join(os.Getenv(constants.BundleDirEnvVar), "lib64", "ld-linux-x86-64.so.2")
	if runtime.GOARCH == "arm64" {
		loader = filepath.Join(os.Getenv(constants.BundleDirEnvVar), "lib", "ld-linux-aarch64.so.1")
	}
	pyBin := filepath.Join(os.Getenv(constants.BundleDirEnvVar), "bin", "python")

	return []string{loader, pyBin}
}

func defaultPythonBinaryArgs(pkgName string) []string {
	return []string{
		"-u",
		"-m",
		pkgName,
	}
}
