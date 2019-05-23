// +build !windows

package main

import (
	"os"
)

func runAgentPlatformSpecific(flags *flags, interruptCh chan os.Signal, exit chan struct{}) {
	runAgent(flags, interruptCh, exit)
}
