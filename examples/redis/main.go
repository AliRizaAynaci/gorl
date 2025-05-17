package main

import (
	"fmt"
	"time"

	"github.com/AliRizaAynaci/gorl"
	"github.com/AliRizaAynaci/gorl/core"
)

func main() {
	limiter, err := gorl.New(core.Config{
		Strategy: core.LeakyBucket,
		KeyBy:    core.KeyByAPIKey,
		Limit:    4,
		Window:   10 * time.Second,
		RedisURL: "redis://localhost:6379/0",
	})
	if err != nil {
		panic(err)
	}

	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow("example-api-key")
		fmt.Printf("Request #%d: allowed=%v, err=%v\n", i+1, allowed, err)
		time.Sleep(1 * time.Second)
	}
}
