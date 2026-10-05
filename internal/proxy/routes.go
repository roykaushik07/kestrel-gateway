// Package proxy loads the route table and reverse-proxies to upstreams.
package proxy

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type RateConfig struct {
	RPS   int `yaml:"rps"`
	Burst int `yaml:"burst"`
}

type Route struct {
	Prefix   string     `yaml:"prefix"`
	Upstream string     `yaml:"upstream"`
	Scopes   []string   `yaml:"scopes"`
	Rate     RateConfig `yaml:"rate"`
}

type SSEConfig struct {
	Upstream         string `yaml:"upstream"`
	Path             string `yaml:"path"`
	HeartbeatSeconds int    `yaml:"heartbeat_seconds"`
}

type Config struct {
	Upstreams map[string]string `yaml:"upstreams"`
	Routes    []Route           `yaml:"routes"`
	SSE       SSEConfig         `yaml:"sse"`
}

// LoadConfig reads and validates the route table.
func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	for _, r := range cfg.Routes {
		if _, ok := cfg.Upstreams[r.Upstream]; !ok {
			return nil, fmt.Errorf("route %s references unknown upstream %q", r.Prefix, r.Upstream)
		}
	}
	return &cfg, nil
}
