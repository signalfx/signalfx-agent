package main

import (
	"expvar"
	_ "expvar"
	"fmt"
	"net"
	"net/http"
)

func init() {
	expvar.Publish("queues", expvar.Func(func() interface{} {
		return map[string]interface{}{
			"count": 5,
			"lengths": []int64{
				4, 2, 1, 0, 5,
			},
		}
	}))
}

func main() {
	sock, err := net.Listen("tcp", "0.0.0.0:8080") //nolint: gosec
	if err != nil {
		panic("Couldn't listen on port 8080")
	}

	fmt.Println("Serving HTTP expvars on http://0.0.0.0:8080/debug/vars")
	_ = http.Serve(sock, nil)
}
