// Package main demonstrates loading resource-scoped limits from a YAML or JSON file.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/AliRizaAynaci/gorl/v2"
	"github.com/AliRizaAynaci/gorl/v2/config"
)

func main() {
	configPath := flag.String("config", "examples/configuration_file/limits.yaml", "path to a GoRL YAML or JSON config")
	flag.Parse()

	cfg, err := config.LoadResourceConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	limiter, err := gorl.NewResourceLimiter(cfg)
	if err != nil {
		log.Fatalf("create limiter: %v", err)
	}
	defer limiter.Close()

	result, err := limiter.AllowResource(context.Background(), "login", "user-123")
	if err != nil {
		log.Fatalf("evaluate limit: %v", err)
	}

	fmt.Printf("allowed=%v limit=%d remaining=%d\n", result.Allowed, result.Limit, result.Remaining)
}
