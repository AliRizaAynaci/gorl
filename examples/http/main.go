// Package main demonstrates using GoRL with the standard net/http middleware.
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/AliRizaAynaci/gorl/v2"
	"github.com/AliRizaAynaci/gorl/v2/core"
	mw "github.com/AliRizaAynaci/gorl/v2/middleware/http"
)

func main() {
	limiter, err := gorl.New(core.Config{
		Strategy: core.SlidingWindow,
		Limit:    5,
		Window:   30 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer limiter.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from %s!\n", r.URL.Path)
	})

	mux := http.NewServeMux()
	mux.Handle("/api/", mw.RateLimit(limiter, mw.Options{
		KeyFunc: mw.KeyByIP(),
	}, handler))

	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
