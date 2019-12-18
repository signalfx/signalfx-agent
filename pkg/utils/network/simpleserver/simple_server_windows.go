// +build windows

package simpleserver

import (
	"net"

	winio "github.com/Microsoft/go-winio"
)

// Listen returns a net.Listener for the specified path
func Listen(path string) (net.Listener, error) {
	return winio.ListenPipe(path, nil)
}

// Dial dials a Named pipe and returns a net.Conn connection
func Dial(path string) (net.Conn, error) {
	return winio.DialPipe(path, nil)
}
