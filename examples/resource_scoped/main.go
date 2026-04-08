// Package main demonstrates using resource-scoped rate limiting with per-resource policies.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/AliRizaAynaci/gorl/v2"
	"github.com/AliRizaAynaci/gorl/v2/core"
)

func main() {
	limiter, err := gorl.NewResourceLimiter(core.ResourceConfig{
		Strategy: core.SlidingWindow,
		DefaultPolicy: core.ResourcePolicy{
			Limit:  100,
			Window: time.Minute,
		},
		Resources: map[string]core.ResourcePolicy{
			"login": {
				Limit:  5,
				Window: time.Minute,
			},
			"search": {
				Limit:  20,
				Window: time.Second,
			},
		},
	})
	if err != nil {
		panic(err)
	}
	defer limiter.Close()

	ctx := context.Background()

	for i := 1; i <= 6; i++ {
		res, err := limiter.AllowResource(ctx, "login", "user-123")
		fmt.Printf("login req %d: allowed=%v remaining=%d err=%v\n", i, res.Allowed, res.Remaining, err)
	}

	for i := 1; i <= 3; i++ {
		res, err := limiter.AllowResource(ctx, "search", "user-123")
		fmt.Printf("search req %d: allowed=%v remaining=%d err=%v\n", i, res.Allowed, res.Remaining, err)
	}

	res, err := limiter.AllowResource(ctx, "unlisted", "user-123")
	fmt.Printf("default resource: allowed=%v remaining=%d err=%v\n", res.Allowed, res.Remaining, err)
}
