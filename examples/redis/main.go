package main

import (
	"fmt"
	"time"

	"github.com/AliRizaAynaci/gorl"
	"github.com/AliRizaAynaci/gorl/core"
)

func main() {
	limiter, err := gorl.New(core.Config{
		Strategy: core.FixedWindow,
		KeyBy:    core.KeyByAPIKey,
		Limit:    4,
		Window:   10 * time.Second,
		RedisURL: "redis://localhost:6379/0",
	})
	if err != nil {
		panic(err)
	}

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
