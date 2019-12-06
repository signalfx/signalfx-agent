package main

import (
	"expvar"
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

	expvar.Publish("memory", expvar.Func(func() interface{} {
		return map[string]interface{}{
			"Allocations": []map[string]int64{
				{"Size": 96, "Mallocs": 64, "Frees": 32,},
				{"Size": 32, "Mallocs": 16, "Frees": 16,},
				{"Size": 64, "Mallocs": 16, "Frees": 48,},
			},
			"HeapAllocation": 96,
		}
	}))
}

func main() {
	var kafkaExJaegerTransactionOk = expvar.NewInt("kafka.ex-jaeger-transaction.ok")
	kafkaExJaegerTransactionOk.Add(11)

	sock, err := net.Listen("tcp", "0.0.0.0:8080") //nolint: gosec
	if err != nil {
		panic("Couldn't listen on port 8080")
	}

	fmt.Println("Serving HTTP expvars on http://0.0.0.0:8080/debug/vars")
	_ = http.Serve(sock, nil)
}
