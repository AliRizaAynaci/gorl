// Package main demonstrates using a custom key extractor with the rate limiter.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/AliRizaAynaci/gorl/v2"
	"github.com/AliRizaAynaci/gorl/v2/core"
)

// main runs a simple demonstration of the rate limiter with a custom key extractor.
func main() {
	customExtractor := func(ctx interface{}) string {
		return ctx.(string)
	}

	limiter, err := gorl.New(core.Config{
		Strategy:           core.LeakyBucket,
		KeyBy:              core.KeyByCustom,
		Limit:              2,
		Window:             5 * time.Second,
		CustomKeyExtractor: customExtractor,
	})
	if err != nil {
		panic(err)
	}
	defer limiter.Close()

	ctx := context.Background()
	users := []string{"user-123", "user-456", "user-123", "user-123"}
	for i, user := range users {
		res, err := limiter.Allow(ctx, user)
		fmt.Printf("Req %d - User: %s, allowed=%v, remaining=%d, err=%v\n", 
			i+1, user, res.Allowed, res.Remaining, err)
	}
}
