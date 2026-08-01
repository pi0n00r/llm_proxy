package config

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

// DefaultMaxRequestBodyBytes permits large prompts and image payloads while
// preventing a single client request from consuming unbounded memory.
const DefaultMaxRequestBodyBytes int64 = 32 << 20

// Config represents the application configuration
type Config struct {
	Server              ServerConfig              `toml:"server"`
	Backend             BackendConfig             `toml:"backend"`
	BackendOpenAI       BackendOpenAIConfig       `toml:"backend_openai"`
	Database            DatabaseConfig            `toml:"database"`
	OllamaCompatibility OllamaCompatibilityConfig `toml:"ollama_compatibility"`
	RequestSanitization RequestSanitizationConfig `toml:"request_sanitization"`
	ChatTextInjection   ChatTextInjectionConfig   `toml:"chat_text_injection"`
	StreamOverride      StreamOverrideConfig      `toml:"stream_override"`
	OllamaOverrides     OllamaOverridesConfig     `toml:"ollama_overrides"`
	Gemma4Fix           Gemma4FixConfig           `toml:"gemma_4_fix"`
}

// ServerConfig holds the server settings
type ServerConfig struct {
	Host                string `toml:"host"`
	Port                int    `toml:"port"`
	MaxRequestBodyBytes int64  `toml:"max_request_body_bytes"`
	EnableCORS          bool   `toml:"enable_cors"`
	LogMessages         bool   `toml:"log_messages"`
	LogRawRequests      bool   `toml:"log_raw_requests"`
	LogRawResponses     bool   `toml:"log_raw_responses"`
	Verbose             bool   `toml:"verbose"`
}

// BackendConfig holds the backend service settings
type BackendConfig struct {
	Type          string   `toml:"type"` // "openai" or "ollama"
	Endpoint      string   `toml:"endpoint"`
	Timeout       int      `toml:"timeout"`        // in seconds
	ToolBlacklist []string `toml:"tool_blacklist"` // List of tool names to filter out
}

// DatabaseConfig holds the database settings
type DatabaseConfig struct {
	Path            string `toml:"path"`
	MaxRequests     int    `toml:"max_requests"`     // Maximum number of requests to keep (0 = unlimited)
	CleanupInterval int    `toml:"cleanup_interval"` // Cleanup interval in minutes (0 = disabled)
	StoreContent    bool   `toml:"store_content"`    // Store prompts and responses in SQLite
}

// OllamaCompatibilityConfig makes server-managed OpenAI backend behavior
// explicit. When a value is configured, matching Ollama hints are accepted and
// removed before translation; conflicting values are rejected.
type OllamaCompatibilityConfig struct {
	ServerManagedKeepAlive string `toml:"server_managed_keep_alive"`
	ServerManagedNumCtx    int    `toml:"server_managed_num_ctx"`
}

// BackendOpenAIConfig holds OpenAI-specific backend settings
type BackendOpenAIConfig struct {
	ForcePromptCache bool `toml:"force_prompt_cache"` // Force prompt caching on all requests
}

// RequestSanitizationConfig holds settings for removing problematic incoming request parameters.
type RequestSanitizationConfig struct {
	MaxTokensPolicy string `toml:"max_tokens_policy"` // "preserve", "drop", or "drop_above"
	MaxTokensLimit  int    `toml:"max_tokens_limit"`  // Used when max_tokens_policy is "drop_above"
}

// ChatTextInjectionConfig holds the chat text injection settings
type ChatTextInjectionConfig struct {
	Enabled bool   `toml:"enabled"` // Enable text injection
	Text    string `toml:"text"`    // Text to inject
	Mode    string `toml:"mode"`    // "first", "last", or "system" - which message to inject into
}

// StreamOverrideConfig holds settings for forcing the streaming behavior of
// requests regardless of what the client asked for.
type StreamOverrideConfig struct {
	Mode string `toml:"mode"` // "passthrough", "always", or "never"
}

// OllamaOverridesConfig holds Ollama-native request overrides. These are
// ignored by the OpenAI backend because it has no equivalent fields.
type OllamaOverridesConfig struct {
	Thinking string `toml:"thinking"` // "passthrough", "off", or "on"
}

// Gemma4FixConfig controls the self-contained mitigation for known Gemma 4 +
// vLLM streaming corruption bugs (leaked tool-call control tokens and leaked
// reasoning-channel tokens in the content field). Only relevant when
// backend.type = "openai". See docs/ for background.
type Gemma4FixConfig struct {
	Enabled bool `toml:"enabled"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	var config Config

	metadata, err := toml.DecodeFile(path, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to read/parse config file: %w", err)
	}

	// Fail on unknown keys
	if len(metadata.Undecoded()) > 0 {
		return nil, fmt.Errorf("unknown keys in config file: %v", metadata.Undecoded())
	}

	// Validate backend type
	if config.Backend.Type != "openai" && config.Backend.Type != "ollama" {
		return nil, fmt.Errorf("invalid backend type: %s (must be 'openai' or 'ollama')", config.Backend.Type)
	}
	if strings.TrimSpace(config.Backend.Endpoint) == "" {
		return nil, fmt.Errorf("backend.endpoint must not be empty")
	}
	if config.Server.Port < 0 || config.Server.Port > 65535 {
		return nil, fmt.Errorf("invalid server.port: %d (must be between 0 and 65535)", config.Server.Port)
	}
	if config.Server.MaxRequestBodyBytes < 0 {
		return nil, fmt.Errorf("invalid server.max_request_body_bytes: %d (must be 0 or greater)", config.Server.MaxRequestBodyBytes)
	}
	if config.Backend.Timeout < 0 {
		return nil, fmt.Errorf("invalid backend.timeout: %d (must be 0 or greater)", config.Backend.Timeout)
	}
	if config.Database.MaxRequests < 0 {
		return nil, fmt.Errorf("invalid database.max_requests: %d (must be 0 or greater)", config.Database.MaxRequests)
	}
	if config.Database.CleanupInterval < 0 {
		return nil, fmt.Errorf("invalid database.cleanup_interval: %d (must be 0 or greater)", config.Database.CleanupInterval)
	}

	// Validate chat text injection mode
	if config.ChatTextInjection.Mode != "" && config.ChatTextInjection.Mode != "first" && config.ChatTextInjection.Mode != "last" && config.ChatTextInjection.Mode != "system" {
		return nil, fmt.Errorf("invalid chat_text_injection.mode: %s (must be 'first', 'last', or 'system')", config.ChatTextInjection.Mode)
	}
	if config.RequestSanitization.MaxTokensPolicy != "" &&
		config.RequestSanitization.MaxTokensPolicy != "preserve" &&
		config.RequestSanitization.MaxTokensPolicy != "drop" &&
		config.RequestSanitization.MaxTokensPolicy != "drop_above" {
		return nil, fmt.Errorf("invalid request_sanitization.max_tokens_policy: %s (must be 'preserve', 'drop', or 'drop_above')", config.RequestSanitization.MaxTokensPolicy)
	}
	if config.RequestSanitization.MaxTokensLimit < 0 {
		return nil, fmt.Errorf("invalid request_sanitization.max_tokens_limit: %d (must be 0 or greater)", config.RequestSanitization.MaxTokensLimit)
	}

	// Validate stream override mode
	if config.StreamOverride.Mode != "" &&
		config.StreamOverride.Mode != "passthrough" &&
		config.StreamOverride.Mode != "always" &&
		config.StreamOverride.Mode != "never" {
		return nil, fmt.Errorf("invalid stream_override.mode: %s (must be 'passthrough', 'always', or 'never')", config.StreamOverride.Mode)
	}
	if config.OllamaOverrides.Thinking != "" &&
		config.OllamaOverrides.Thinking != "passthrough" &&
		config.OllamaOverrides.Thinking != "off" &&
		config.OllamaOverrides.Thinking != "on" {
		return nil, fmt.Errorf("invalid ollama_overrides.thinking: %s (must be 'passthrough', 'off', or 'on')", config.OllamaOverrides.Thinking)
	}
	if config.OllamaCompatibility.ServerManagedNumCtx < 0 {
		return nil, fmt.Errorf("invalid ollama_compatibility.server_managed_num_ctx: %d (must be 0 or greater)", config.OllamaCompatibility.ServerManagedNumCtx)
	}

	// Set defaults
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 11434
	}
	if config.Server.MaxRequestBodyBytes == 0 {
		config.Server.MaxRequestBodyBytes = DefaultMaxRequestBodyBytes
	}
	if config.Backend.Timeout == 0 {
		config.Backend.Timeout = 300
	}
	if config.Database.Path == "" {
		config.Database.Path = "./llm_proxy.db"
	}
	if !metadata.IsDefined("database", "max_requests") {
		config.Database.MaxRequests = 100
	}
	if !metadata.IsDefined("database", "cleanup_interval") {
		config.Database.CleanupInterval = 5
	}
	if !metadata.IsDefined("database", "store_content") {
		config.Database.StoreContent = true
	}
	if config.ChatTextInjection.Mode == "" {
		config.ChatTextInjection.Mode = "last"
	}
	if config.RequestSanitization.MaxTokensPolicy == "" {
		config.RequestSanitization.MaxTokensPolicy = "preserve"
	}
	if config.StreamOverride.Mode == "" {
		config.StreamOverride.Mode = "passthrough"
	}
	if config.OllamaOverrides.Thinking == "" {
		config.OllamaOverrides.Thinking = "passthrough"
	}

	return &config, nil
}
