package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/signalfx/signalfx-go-tracing/tracing"
)

func main() {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()
	tracing.Start(tracing.WithEndpointURL("http://signalfx-agent:9080/v1/trace"), tracing.WithServiceName("tracing"))
	defer tracing.Stop()
	counterGauge := promauto.NewCounter(prometheus.CounterOpts{
		Name: "counter",
		Help: "The total number of times we produced a trace",
	})
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
			counterGauge.Inc()
			break
		case <-c:
			ticker.Stop()
			return
		}
	}

}
