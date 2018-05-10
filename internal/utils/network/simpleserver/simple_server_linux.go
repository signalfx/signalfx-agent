// +build !windows

package simpleserver

import (
	"net"
)

// Listen returns a net.Listener for the specified path
func Listen(path string) (net.Listener, error) {
	return net.Listen("unix", path)
}

// Dial dials a UNIX domain socket and returns a net.Conn connection
func Dial(path string) (net.Conn, error) {
	return net.Dial("unix", path)
}
