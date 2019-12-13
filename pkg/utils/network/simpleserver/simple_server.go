package simpleserver

import (
	"net"
	"os"
)

// Run creates and runs a simple server that will call handler for each
// connection and write back whatever that function returns to the client.
// Content is served over a Named Pipe on Windows or a Unix domain socket on
// Linux and Mac.
// Returns a function that can be called to stop the server.
func Run(path string, handler func(net.Conn) string, errs func(error)) (func(), error) {
	closed := false
	os.Remove(path)

	listener, err := Listen(path)

	if err != nil {
		return nil, err
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if closed {
				return
			}
			if err != nil {
				errs(err)
				continue
			}

			_, err = conn.Write([]byte(handler(conn)))
			if err != nil {
				errs(err)
			}
			conn.Close()
		}
	}()

	return func() {
		closed = true
		listener.Close()
	}, nil
}
