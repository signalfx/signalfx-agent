// +build windows

package pyrunner

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
)

// The Windows specific process attributes
func procAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		//Pdeathsig: syscall.SIGTERM,
	}
}

func pythonBinaryExecutable() string {
	return filepath.Join(os.Getenv(constants.BundleDirEnvVar), "python", "python.exe")
}

func pythonBinaryArgs(pkgName string) []string {
	return []string{
		"-m",
		pkgName,
	}
}
