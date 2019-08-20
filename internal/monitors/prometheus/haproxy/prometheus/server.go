package exporter

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func StartServer(conf *Config, ctx context.Context) {
	go func() {
		var (
			exporter *Exporter
			versionCollector prometheus.Collector
			procExporter prometheus.Collector
			)
		if strings.TrimSpace(conf.ServerMetricFields) == "" { conf.ServerMetricFields = serverMetrics.String() }
		selectedServerMetrics, err := filterServerMetrics(conf.ServerMetricFields)
		if err != nil { log.Fatal(err) }
		exporter, err = NewExporter(conf.ScrapeURI, *conf.SSLVerify, selectedServerMetrics, time.Duration(*conf.TimeoutSeconds) * time.Second)
		if err != nil { log.Fatal(err) }
		prometheus.MustRegister(exporter)
		versionCollector = version.NewCollector("haproxy_exporter")
		prometheus.MustRegister(versionCollector)
		if conf.PidFile != "" {
			procExporter = prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{
				PidFn: func() (int, error) {
					content, err := ioutil.ReadFile(conf.PidFile)
					if err != nil { return 0, fmt.Errorf("can't read pid file: %s", err) }
					value, err := strconv.Atoi(strings.TrimSpace(string(content)))
					if err != nil { return 0, fmt.Errorf("can't parse pid file: %s", err) }
					return value, nil
				},
				Namespace: namespace,
			})
			prometheus.MustRegister(procExporter)
		}
		log.Infoln("Listening on", conf.ListenAddress)
		http.Handle(conf.MetricsPath, promhttp.Handler())
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<html><head><title>Haproxy Exporter</title></head><body><h1>Haproxy Exporter</h1><p><a href='` + conf.MetricsPath + `'>Metrics</a></p></body></html>`))
		})
		log.Fatal(http.ListenAndServe(conf.ListenAddress, nil))
		//for {
		//	select {
     	//		case <-ctx.Done():
		//			if exporter         != nil { prometheus.Unregister(exporter) }
		//			if versionCollector != nil { prometheus.Unregister(versionCollector) }
		//			if procExporter     != nil { prometheus.Unregister(procExporter) }
		//			fmt.Printf("prometheus.Unregister: %+v", exporter)
		//			return
		//		default:
		//	}
		//}
	}()
}
