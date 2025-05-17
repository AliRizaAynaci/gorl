package main

import (
	"fmt"
	"time"

	"github.com/AliRizaAynaci/gorl"
	"github.com/AliRizaAynaci/gorl/core"
)

func main() {
	limiter, _ := gorl.New(core.Config{
		Strategy: core.LeakyBucket,
		KeyBy:    core.KeyByIP,
		Limit:    3,
		Window:   5 * time.Second,
	})

	for i := 1; i <= 15; i++ {
		allowed, err := limiter.Allow("127.0.0.1")
		fmt.Printf("Request #%d: allowed=%v, err=%v\n", i, allowed, err)
		time.Sleep(500 * time.Millisecond)
	}

}
