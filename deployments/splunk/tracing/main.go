package main

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/signalfx/signalfx-go-tracing/tracing"
)

func main() {
	tracing.Start(tracing.WithEndpointURL("http://signalfx-agent:9080/v1/trace"), tracing.WithServiceName("tracing"))
	defer tracing.Stop()
	counter := 0
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			span := opentracing.GlobalTracer().StartSpan("main")
			span.SetTag("span.kind", "server")
			span.SetTag("counter", strconv.Itoa(counter))
			childSpan := opentracing.GlobalTracer().StartSpan("sub1", opentracing.ChildOf(span.Context()))
			time.Sleep(1 * time.Second)
			childSpan.Finish()
			span.Finish()
			counter++
			break
		case <-c:
			ticker.Stop()
			return
		}
	}

}
