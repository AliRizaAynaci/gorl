// Package main demonstrates the usage of the in-memory rate limiter.
package main

import (
	"fmt"
	"time"

	"github.com/AliRizaAynaci/gorl"
	"github.com/AliRizaAynaci/gorl/core"
)

// main runs a simple demonstration of the rate limiter.
func main() {
	limiter, _ := gorl.New(core.Config{
		Strategy: core.TokenBucket,
		KeyBy:    core.KeyByIP,
		Limit:    3,
		Window:   10 * time.Second,
	})

	start := time.Now()
	for i := 1; i <= 15; i++ {
		allowed, err := limiter.Allow("127.0.0.1")
		elapsed := time.Since(start).Seconds()
		timestamp := time.Now().Format("15:04:05")

		fmt.Printf("[%s +%.1fs] Request #%d: allowed=%v, err=%v\n",
			timestamp, elapsed, i, allowed, err,
		)

		time.Sleep(1000 * time.Millisecond)
	}
}
