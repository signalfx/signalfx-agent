// +build !windows,!linux

package vmem

import (
	"context"
	"fmt"
)

// Configure is the main function of the monitor, it will report host metadata
// on a varied interval
func (m *Monitor) Configure(conf *Config) error {
	return fmt.Errorf("this monitor is not implemented on this platform")
}

func (m *Monitor) Collect(ctx context.Context) error {

}
