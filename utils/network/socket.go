package network

import (
	"net"
	"os"
)

func RunSimpleSocketServer(path string, handler func(net.Conn) string, errs func(error)) (func(), error) {
	closed := false

	os.Remove(path)
	sock, err := net.Listen("unix", path)

	if err != nil {
		return nil, err
	}

	go func() {
		for {
			conn, err := sock.Accept()
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
		sock.Close()
	}, nil
}
