package handlers

import (
	"testing"

	"llm_proxy/config"
)

func TestFrontendURLBracketsIPv6(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{Host: "::", Port: 6666}}
	if got, want := frontendURL(cfg, "/api/chat"), "http://[::]:6666/api/chat"; got != want {
		t.Fatalf("frontendURL() = %q, want %q", got, want)
	}
}
