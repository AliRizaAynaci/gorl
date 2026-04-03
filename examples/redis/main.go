// Package main demonstrates the usage of the Redis-backed rate limiter.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/AliRizaAynaci/gorl/v2"
	"github.com/AliRizaAynaci/gorl/v2/core"
)

// main runs a simple demonstration of the rate limiter using a Redis backend.
func main() {
	limiter, err := gorl.New(core.Config{
		Strategy: core.TokenBucket,
		Limit:    4,
		Window:   10 * time.Second,
		RedisURL: "redis://localhost:6379/0",
		FailOpen: true,
	})
	if err != nil {
		panic(err)
	}
	defer limiter.Close()

	ctx := context.Background()
	start := time.Now()
	for i := 1; i <= 15; i++ {
		res, err := limiter.Allow(ctx, "127.0.0.1")
		elapsed := time.Since(start).Seconds()
		timestamp := time.Now().Format("15:04:05")

		fmt.Printf("[%s +%.1fs] Request #%d: allowed=%v, remaining=%d, retry_after=%v, err=%v\n",
			timestamp, elapsed, i, res.Allowed, res.Remaining, res.RetryAfter, err,
		)

		time.Sleep(1000 * time.Millisecond)
	}
}
