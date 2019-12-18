package main

import (
	"expvar"
	"fmt"
	"net"
	"net/http"
)

func init() {
	expvar.Publish("memory", expvar.Func(func() interface{} {
		return map[string]interface{}{
			"Allocations": []map[string]int64{
				{"Size": 96, "Mallocs": 64, "Frees": 32},
				{"Size": 32, "Mallocs": 16, "Frees": 16},
				{"Size": 64, "Mallocs": 16, "Frees": 48},
			},
			"HeapAllocation": 96,
		}
	}))
}

func main() {
	expvar.Publish("queues", expvar.Func(func() interface{} {
		return map[string]interface{}{
			"count": 5,
			"lengths": []int64{
				4, 2, 1, 0, 5,
			},
		}
	}))
	expvar.NewInt("kafka.ex-jaeger-transaction.ok").Add(11)
	expvar.NewInt("willplayad.in_flight").Set(0)
	expvar.Publish("willplayad.response.noserv", expvar.Func(func() interface{} { return struct{}{} }))
	expvar.NewInt("willplayad.response.serv").Set(0)
	expvar.NewInt("willplayad.start").Set(0)

	sock, err := net.Listen("tcp", "0.0.0.0:8080") //nolint: gosec
	if err != nil {
		panic("Couldn't listen on port 8080")
	}

	fmt.Println("Serving HTTP expvars on http://0.0.0.0:8080/debug/vars")
	_ = http.Serve(sock, nil)
}
