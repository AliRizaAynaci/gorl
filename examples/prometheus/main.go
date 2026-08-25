// Package main demonstrates exporting GoRL metrics with Prometheus.
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/AliRizaAynaci/gorl/v2"
	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/AliRizaAynaci/gorl/v2/metrics"
	mw "github.com/AliRizaAynaci/gorl/v2/middleware/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	collector := metrics.NewPrometheusCollector("gorl", "example")
	metrics.RegisterPrometheusCollectors(collector)

	limiter, err := gorl.New(core.Config{
		Strategy: core.SlidingWindow,
		Limit:    5,
		Window:   time.Minute,
		Metrics:  collector,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer limiter.Close()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/api/", mw.RateLimit(limiter, mw.Options{
		KeyFunc: mw.KeyByIP(),
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "allowed")
	})))

	log.Println("API: http://localhost:8080/api/")
	log.Println("metrics: http://localhost:8080/metrics")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
