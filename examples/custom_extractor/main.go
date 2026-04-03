// Package main demonstrates building a custom application key before calling the limiter.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/AliRizaAynaci/gorl/v2"
	"github.com/AliRizaAynaci/gorl/v2/core"
)

// main runs a simple demonstration of custom key generation in application code.
func main() {
	limiter, err := gorl.New(core.Config{
		Strategy: core.LeakyBucket,
		Limit:    2,
		Window:   5 * time.Second,
	})
	if err != nil {
		panic(err)
	}
	defer limiter.Close()

	ctx := context.Background()
	tenantUsers := []struct {
		tenant string
		user   string
	}{
		{tenant: "team-a", user: "user-123"},
		{tenant: "team-b", user: "user-456"},
		{tenant: "team-a", user: "user-123"},
		{tenant: "team-a", user: "user-123"},
	}

	for i, item := range tenantUsers {
		key := item.tenant + ":" + item.user
		res, err := limiter.Allow(ctx, key)
		fmt.Printf("Req %d - Key: %s, allowed=%v, remaining=%d, err=%v\n",
			i+1, key, res.Allowed, res.Remaining, err)
	}
}
