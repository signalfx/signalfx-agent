// +build !windows

package agentmain

import (
	"os"
)

func runAgentPlatformSpecific(flags *flags, interruptCh chan os.Signal, exit chan struct{}) {
	runAgent(flags, interruptCh, exit)
}
