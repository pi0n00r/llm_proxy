package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRequestSanitizationConfig(t *testing.T) {
	path := writeTestConfig(t, `
[backend]
type = "openai"
endpoint = "http://localhost:8008"

[request_sanitization]
max_tokens_policy = "drop_above"
max_tokens_limit = 8192
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.RequestSanitization.MaxTokensPolicy != "drop_above" {
		t.Fatalf("MaxTokensPolicy = %q, want drop_above", cfg.RequestSanitization.MaxTokensPolicy)
	}
	if cfg.RequestSanitization.MaxTokensLimit != 8192 {
		t.Fatalf("MaxTokensLimit = %d, want 8192", cfg.RequestSanitization.MaxTokensLimit)
	}
}

func TestLoadDefaultsRequestSanitizationPolicy(t *testing.T) {
	path := writeTestConfig(t, `
[backend]
type = "openai"
endpoint = "http://localhost:8008"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.RequestSanitization.MaxTokensPolicy != "preserve" {
		t.Fatalf("MaxTokensPolicy = %q, want preserve", cfg.RequestSanitization.MaxTokensPolicy)
	}
}

func TestLoadRejectsInvalidMaxTokensPolicy(t *testing.T) {
	path := writeTestConfig(t, `
[backend]
type = "openai"
endpoint = "http://localhost:8008"

[request_sanitization]
max_tokens_policy = "clamp"
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "request_sanitization.max_tokens_policy") {
		t.Fatalf("Load() error = %v, want max_tokens_policy error", err)
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	path := writeTestConfig(t, `
[backend]
type = "openai"
endpoint = "http://localhost:8008"
unexpected = true
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "unknown keys") {
		t.Fatalf("Load() error = %v, want unknown keys error", err)
	}
}

func TestLoadRejectsInvalidChatTextInjectionMode(t *testing.T) {
	path := writeTestConfig(t, `
[backend]
type = "openai"
endpoint = "http://localhost:8008"

[chat_text_injection]
mode = "middle"
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "chat_text_injection.mode") {
		t.Fatalf("Load() error = %v, want chat_text_injection.mode error", err)
	}
}

func TestLoadStreamOverrideConfig(t *testing.T) {
	path := writeTestConfig(t, `
[backend]
type = "openai"
endpoint = "http://localhost:8008"

[stream_override]
mode = "always"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.StreamOverride.Mode != "always" {
		t.Fatalf("StreamOverride.Mode = %q, want always", cfg.StreamOverride.Mode)
	}
}

func TestLoadDefaultsStreamOverrideMode(t *testing.T) {
	path := writeTestConfig(t, `
[backend]
type = "openai"
endpoint = "http://localhost:8008"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.StreamOverride.Mode != "passthrough" {
		t.Fatalf("StreamOverride.Mode = %q, want passthrough", cfg.StreamOverride.Mode)
	}
}

func TestLoadRejectsInvalidStreamOverrideMode(t *testing.T) {
	path := writeTestConfig(t, `
[backend]
type = "openai"
endpoint = "http://localhost:8008"

[stream_override]
mode = "sometimes"
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "stream_override.mode") {
		t.Fatalf("Load() error = %v, want stream_override.mode error", err)
	}
}

func TestLoadOllamaThinkingOverrideConfig(t *testing.T) {
	path := writeTestConfig(t, `
[backend]
type = "ollama"
endpoint = "http://localhost:11434"

[ollama_overrides]
thinking = "off"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.OllamaOverrides.Thinking != "off" {
		t.Fatalf("OllamaOverrides.Thinking = %q, want off", cfg.OllamaOverrides.Thinking)
	}
}

func TestLoadDefaultsOllamaThinkingOverrideMode(t *testing.T) {
	path := writeTestConfig(t, `
[backend]
type = "ollama"
endpoint = "http://localhost:11434"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.OllamaOverrides.Thinking != "passthrough" {
		t.Fatalf("OllamaOverrides.Thinking = %q, want passthrough", cfg.OllamaOverrides.Thinking)
	}
}

func TestLoadRejectsInvalidOllamaThinkingOverrideMode(t *testing.T) {
	path := writeTestConfig(t, `
[backend]
type = "ollama"
endpoint = "http://localhost:11434"

[ollama_overrides]
thinking = "false"
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "ollama_overrides.thinking") {
		t.Fatalf("Load() error = %v, want ollama_overrides.thinking error", err)
	}
}

func TestLoadAppliesOperationalDefaults(t *testing.T) {
	path := writeTestConfig(t, `
[backend]
type = "ollama"
endpoint = "http://localhost:11434"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Fatalf("Server.Host = %q, want 0.0.0.0", cfg.Server.Host)
	}
	if cfg.Server.Port != 11434 {
		t.Fatalf("Server.Port = %d, want 11434", cfg.Server.Port)
	}
	if cfg.Server.MaxRequestBodyBytes != DefaultMaxRequestBodyBytes {
		t.Fatalf("Server.MaxRequestBodyBytes = %d, want %d", cfg.Server.MaxRequestBodyBytes, DefaultMaxRequestBodyBytes)
	}
	if cfg.Backend.Timeout != 300 {
		t.Fatalf("Backend.Timeout = %d, want 300", cfg.Backend.Timeout)
	}
	if cfg.Database.Path != "./llm_proxy.db" {
		t.Fatalf("Database.Path = %q, want ./llm_proxy.db", cfg.Database.Path)
	}
	if cfg.Database.MaxRequests != 100 || cfg.Database.CleanupInterval != 5 {
		t.Fatalf("database cleanup defaults = (%d, %d), want (100, 5)", cfg.Database.MaxRequests, cfg.Database.CleanupInterval)
	}
	if cfg.ChatTextInjection.Mode != "last" {
		t.Fatalf("ChatTextInjection.Mode = %q, want last", cfg.ChatTextInjection.Mode)
	}
}

func TestLoadPreservesExplicitDatabaseCleanupZeros(t *testing.T) {
	path := writeTestConfig(t, `
[backend]
type = "openai"
endpoint = "http://localhost:8008"

[database]
max_requests = 0
cleanup_interval = 0
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.MaxRequests != 0 || cfg.Database.CleanupInterval != 0 {
		t.Fatalf("database cleanup values = (%d, %d), want explicit zeros", cfg.Database.MaxRequests, cfg.Database.CleanupInterval)
	}
}

func TestLoadServerRequestBodyLimit(t *testing.T) {
	path := writeTestConfig(t, `
[server]
max_request_body_bytes = 1048576

[backend]
type = "openai"
endpoint = "http://localhost:8008"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.MaxRequestBodyBytes != 1048576 {
		t.Fatalf("Server.MaxRequestBodyBytes = %d, want 1048576", cfg.Server.MaxRequestBodyBytes)
	}
}

func TestLoadRejectsInvalidOperationalNumbers(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "negative port",
			content: `
[server]
port = -1
[backend]
type = "openai"
endpoint = "http://localhost:8008"
`,
			want: "server.port",
		},
		{
			name: "port above range",
			content: `
[server]
port = 65536
[backend]
type = "openai"
endpoint = "http://localhost:8008"
`,
			want: "server.port",
		},
		{
			name: "negative request limit",
			content: `
[server]
max_request_body_bytes = -1
[backend]
type = "openai"
endpoint = "http://localhost:8008"
`,
			want: "server.max_request_body_bytes",
		},
		{
			name: "negative backend timeout",
			content: `
[backend]
type = "openai"
endpoint = "http://localhost:8008"
timeout = -1
`,
			want: "backend.timeout",
		},
		{
			name: "negative maximum requests",
			content: `
[backend]
type = "openai"
endpoint = "http://localhost:8008"
[database]
max_requests = -1
`,
			want: "database.max_requests",
		},
		{
			name: "negative cleanup interval",
			content: `
[backend]
type = "openai"
endpoint = "http://localhost:8008"
[database]
cleanup_interval = -1
`,
			want: "database.cleanup_interval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(writeTestConfig(t, tt.content))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Load() error = %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestLoadGemma4FixConfig(t *testing.T) {
	path := writeTestConfig(t, `
[backend]
type = "openai"
endpoint = "http://localhost:8008"

[gemma_4_fix]
enabled = true
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Gemma4Fix.Enabled {
		t.Fatal("Gemma4Fix.Enabled = false, want true")
	}
}

func TestLoadDefaultsGemma4FixDisabled(t *testing.T) {
	path := writeTestConfig(t, `
[backend]
type = "openai"
endpoint = "http://localhost:8008"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Gemma4Fix.Enabled {
		t.Fatal("Gemma4Fix.Enabled = true, want false (default off)")
	}
}

func TestLoadFleetCompatibilityAndContentStorage(t *testing.T) {
	path := writeTestConfig(t, `
[backend]
type = "openai"
endpoint = "http://localhost:8080"

[database]
store_content = false

[ollama_compatibility]
server_managed_keep_alive = "-1"
server_managed_num_ctx = 131072
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.StoreContent {
		t.Fatal("Database.StoreContent = true, want false")
	}
	if cfg.OllamaCompatibility.ServerManagedKeepAlive != "-1" ||
		cfg.OllamaCompatibility.ServerManagedNumCtx != 131072 {
		t.Fatalf("OllamaCompatibility = %#v", cfg.OllamaCompatibility)
	}
}

func TestLoadDefaultsContentStorageEnabled(t *testing.T) {
	path := writeTestConfig(t, `
[backend]
type = "openai"
endpoint = "http://localhost:8080"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Database.StoreContent {
		t.Fatal("Database.StoreContent = false, want backward-compatible true default")
	}
}

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return path
}
