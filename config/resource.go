package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/goccy/go-yaml"
)

type resourceConfigDocument struct {
	Strategy  core.StrategyType                 `json:"strategy" yaml:"strategy"`
	RedisURL  string                            `json:"redis_url" yaml:"redis_url"`
	FailOpen  bool                              `json:"fail_open" yaml:"fail_open"`
	Default   resourcePolicyDocument            `json:"default" yaml:"default"`
	Resources map[string]resourcePolicyDocument `json:"resources" yaml:"resources"`
}

type resourceConfigEnvelope struct {
	GoRL *resourceConfigDocument `json:"gorl" yaml:"gorl"`
}

type resourcePolicyDocument struct {
	Limit  int    `json:"limit" yaml:"limit"`
	Window string `json:"window" yaml:"window"`
}

// LoadResourceConfig loads a resource-scoped limiter configuration from a JSON or YAML file.
func LoadResourceConfig(path string) (core.ResourceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return core.ResourceConfig{}, fmt.Errorf("read config: %w", err)
	}

	var doc resourceConfigDocument
	var envelope resourceConfigEnvelope
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		if err := json.Unmarshal(data, &doc); err != nil {
			return core.ResourceConfig{}, fmt.Errorf("parse json config: %w", err)
		}
		if err := json.Unmarshal(data, &envelope); err != nil {
			return core.ResourceConfig{}, fmt.Errorf("parse json config envelope: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return core.ResourceConfig{}, fmt.Errorf("parse yaml config: %w", err)
		}
		if err := yaml.Unmarshal(data, &envelope); err != nil {
			return core.ResourceConfig{}, fmt.Errorf("parse yaml config envelope: %w", err)
		}
	default:
		return core.ResourceConfig{}, fmt.Errorf("unsupported config format %q", filepath.Ext(path))
	}

	if envelope.GoRL != nil {
		doc = *envelope.GoRL
	}

	return doc.toCore()
}

func (d resourceConfigDocument) toCore() (core.ResourceConfig, error) {
	defaultPolicy, err := d.Default.toCore("default")
	if err != nil {
		return core.ResourceConfig{}, err
	}

	resources := make(map[string]core.ResourcePolicy, len(d.Resources))
	for resource, policyDoc := range d.Resources {
		policy, err := policyDoc.toCore(fmt.Sprintf("resource %q", resource))
		if err != nil {
			return core.ResourceConfig{}, err
		}
		resources[resource] = policy
	}

	cfg := core.ResourceConfig{
		Strategy:      d.Strategy,
		DefaultPolicy: defaultPolicy,
		Resources:     resources,
		RedisURL:      d.RedisURL,
		FailOpen:      d.FailOpen,
	}

	if err := cfg.Validate(); err != nil {
		return core.ResourceConfig{}, err
	}
	return cfg, nil
}

func (p resourcePolicyDocument) toCore(label string) (core.ResourcePolicy, error) {
	window, err := time.ParseDuration(p.Window)
	if err != nil {
		return core.ResourcePolicy{}, fmt.Errorf("%s window: %w", label, err)
	}
	return core.ResourcePolicy{
		Limit:  p.Limit,
		Window: window,
	}, nil
}
