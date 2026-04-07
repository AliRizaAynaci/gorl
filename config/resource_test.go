package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
)

func TestLoadResourceConfig_JSON(t *testing.T) {
	path := writeTempConfig(t, "resource-config.json", `{
  "strategy": "sliding_window",
  "redis_url": "redis://localhost:6379/0",
  "fail_open": true,
  "default": {
    "limit": 100,
    "window": "1m"
  },
  "resources": {
    "login": {
      "limit": 5,
      "window": "1m"
    },
    "search": {
      "limit": 50,
      "window": "1s"
    }
  }
}`)

	cfg, err := LoadResourceConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Strategy != core.SlidingWindow {
		t.Fatalf("expected sliding_window, got %q", cfg.Strategy)
	}
	if cfg.RedisURL != "redis://localhost:6379/0" {
		t.Fatalf("unexpected redis url: %q", cfg.RedisURL)
	}
	if !cfg.FailOpen {
		t.Fatal("expected fail_open=true")
	}
	if cfg.DefaultPolicy.Limit != 100 || cfg.DefaultPolicy.Window != time.Minute {
		t.Fatalf("unexpected default policy: %+v", cfg.DefaultPolicy)
	}
	if cfg.Resources["login"].Limit != 5 {
		t.Fatalf("unexpected login limit: %+v", cfg.Resources["login"])
	}
	if cfg.Resources["search"].Window != time.Second {
		t.Fatalf("unexpected search window: %+v", cfg.Resources["search"])
	}
}

func TestLoadResourceConfig_YAML(t *testing.T) {
	path := writeTempConfig(t, "resource-config.yaml", `
strategy: token_bucket
default:
  limit: 20
  window: 30s
resources:
  outbound-api:
    limit: 10
    window: 1s
`)

	cfg, err := LoadResourceConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Strategy != core.TokenBucket {
		t.Fatalf("expected token_bucket, got %q", cfg.Strategy)
	}
	if cfg.DefaultPolicy.Window != 30*time.Second {
		t.Fatalf("unexpected default window: %v", cfg.DefaultPolicy.Window)
	}
	if cfg.Resources["outbound-api"].Limit != 10 {
		t.Fatalf("unexpected named resource policy: %+v", cfg.Resources["outbound-api"])
	}
}

func TestLoadResourceConfig_InvalidDuration(t *testing.T) {
	path := writeTempConfig(t, "resource-config.yaml", `
strategy: fixed_window
default:
  limit: 10
  window: nope
`)

	if _, err := LoadResourceConfig(path); err == nil {
		t.Fatal("expected invalid duration error")
	}
}

func TestLoadResourceConfig_UnsupportedExtension(t *testing.T) {
	path := writeTempConfig(t, "resource-config.txt", "hello")

	if _, err := LoadResourceConfig(path); err == nil {
		t.Fatal("expected unsupported format error")
	}
}

func TestLoadResourceConfig_MissingDefaultPolicy(t *testing.T) {
	path := writeTempConfig(t, "resource-config.json", `{
  "strategy": "fixed_window",
  "resources": {
    "login": {
      "limit": 5,
      "window": "1s"
    }
  }
}`)

	if _, err := LoadResourceConfig(path); err == nil {
		t.Fatal("expected missing default policy error")
	}
}

func writeTempConfig(t *testing.T, name, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}
