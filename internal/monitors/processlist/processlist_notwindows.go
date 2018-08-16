// +build !windows

package processlist

import "bytes"

// ProcessList takes a snapshot of running processes and returns a byte buffer
func ProcessList() (*bytes.Buffer, error) {
	return &bytes.Buffer{}, nil
}
