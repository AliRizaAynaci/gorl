// Package main demonstrates using a custom key extractor with the rate limiter.
package main

import (
	"fmt"
	"time"

	"github.com/AliRizaAynaci/gorl"
	"github.com/AliRizaAynaci/gorl/core"
)

// main runs a simple demonstration of the rate limiter with a custom key extractor.
func main() {
	customExtractor := func(ctx interface{}) string {
		return ctx.(string)
	}

	limiter, _ := gorl.New(core.Config{
		Strategy:           core.LeakyBucket,
		KeyBy:              core.KeyByCustom,
		Limit:              2,
		Window:             5 * time.Second,
		CustomKeyExtractor: customExtractor,
	})

	users := []string{"user-123", "user-456", "user-123", "user-123"}
	for i, user := range users {
		allowed, err := limiter.Allow(user)
		fmt.Printf("Req %d - User: %s, allowed=%v, err=%v\n", i+1, user, allowed, err)
	}
}
