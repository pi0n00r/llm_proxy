package handlers

import (
	"strings"
	"testing"

	"llm_proxy/config"
	"llm_proxy/models"
)

func fleetCompatibilityConfig() *config.Config {
	return &config.Config{
		Backend: config.BackendConfig{Type: "openai"},
		OllamaCompatibility: config.OllamaCompatibilityConfig{
			ServerManagedKeepAlive: "-1",
			ServerManagedNumCtx:    131072,
		},
	}
}

func TestServerManagedChatCompatibilityAcceptsAndDropsMatchingHints(t *testing.T) {
	req := models.ChatRequest{
		KeepAlive: "-1",
		Options:   map[string]interface{}{"num_ctx": float64(131072), "temperature": 0.2},
	}
	if err := applyServerManagedChatCompatibility(&req, fleetCompatibilityConfig()); err != nil {
		t.Fatalf("applyServerManagedChatCompatibility() error = %v", err)
	}
	if req.KeepAlive != "" {
		t.Fatalf("KeepAlive = %q, want removed", req.KeepAlive)
	}
	if _, ok := req.Options["num_ctx"]; ok {
		t.Fatal("Options retained server-managed num_ctx")
	}
	if req.Options["temperature"] != 0.2 {
		t.Fatalf("unrelated option changed: %#v", req.Options)
	}
}

func TestServerManagedCompatibilityRejectsConflicts(t *testing.T) {
	tests := []struct {
		name string
		req  models.ChatRequest
		want string
	}{
		{
			name: "keep alive",
			req:  models.ChatRequest{KeepAlive: "5m"},
			want: "conflicts with server-managed value",
		},
		{
			name: "context",
			req:  models.ChatRequest{Options: map[string]interface{}{"num_ctx": float64(32768)}},
			want: "conflicts with server-managed value",
		},
		{
			name: "fractional context",
			req:  models.ChatRequest{Options: map[string]interface{}{"num_ctx": 131072.5}},
			want: "positive integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := applyServerManagedChatCompatibility(&tt.req, fleetCompatibilityConfig())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestServerManagedCompatibilityRequiresExplicitConfiguration(t *testing.T) {
	cfg := &config.Config{Backend: config.BackendConfig{Type: "openai"}}
	req := models.GenerateRequest{
		KeepAlive: "-1",
		Options:   map[string]interface{}{"num_ctx": float64(131072)},
	}
	err := applyServerManagedGenerateCompatibility(&req, cfg)
	if err == nil || !strings.Contains(err.Error(), "configure ollama_compatibility") {
		t.Fatalf("error = %v, want explicit configuration error", err)
	}
}

func TestServerManagedCompatibilityDoesNotAlterOllamaBackend(t *testing.T) {
	cfg := fleetCompatibilityConfig()
	cfg.Backend.Type = "ollama"
	req := models.ChatRequest{
		KeepAlive: "5m",
		Options:   map[string]interface{}{"num_ctx": float64(32768)},
	}
	if err := applyServerManagedChatCompatibility(&req, cfg); err != nil {
		t.Fatalf("applyServerManagedChatCompatibility() error = %v", err)
	}
	if req.KeepAlive != "5m" || req.Options["num_ctx"] != float64(32768) {
		t.Fatalf("Ollama request changed: %#v", req)
	}
}
