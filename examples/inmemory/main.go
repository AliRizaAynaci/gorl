package main

import (
	"fmt"
	"time"

	"github.com/AliRizaAynaci/gorl"
	"github.com/AliRizaAynaci/gorl/core"
)

func main() {
	limiter, _ := gorl.New(core.Config{
		Strategy: core.TokenBucket,
		KeyBy:    core.KeyByIP,
		Limit:    5,
		Window:   10 * time.Second,
	})

	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow("127.0.0.1")
		fmt.Printf("Request #%d: allowed=%v, err=%v\n", i+1, allowed, err)
		time.Sleep(1 * time.Second)
	}
}
