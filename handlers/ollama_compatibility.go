package handlers

import (
	"fmt"
	"math"

	"llm_proxy/config"
	"llm_proxy/models"
)

func applyServerManagedChatCompatibility(req *models.ChatRequest, cfg *config.Config) error {
	if cfg.Backend.Type != "openai" {
		return nil
	}
	if err := validateServerManagedKeepAlive(req.KeepAlive, cfg); err != nil {
		return err
	}
	req.KeepAlive = ""

	if err := validateAndDropServerManagedNumCtx(req.Options, cfg); err != nil {
		return err
	}
	return nil
}

func applyServerManagedGenerateCompatibility(req *models.GenerateRequest, cfg *config.Config) error {
	if cfg.Backend.Type != "openai" {
		return nil
	}
	if err := validateServerManagedKeepAlive(req.KeepAlive, cfg); err != nil {
		return err
	}
	req.KeepAlive = ""

	if err := validateAndDropServerManagedNumCtx(req.Options, cfg); err != nil {
		return err
	}
	return nil
}

func validateServerManagedKeepAlive(value string, cfg *config.Config) error {
	if value == "" {
		return nil
	}
	expected := cfg.OllamaCompatibility.ServerManagedKeepAlive
	if expected == "" {
		return fmt.Errorf("keep_alive is unsupported by the OpenAI backend; configure ollama_compatibility.server_managed_keep_alive to acknowledge server-managed model residence")
	}
	if value != expected {
		return fmt.Errorf("keep_alive=%q conflicts with server-managed value %q", value, expected)
	}
	return nil
}

func validateAndDropServerManagedNumCtx(options map[string]interface{}, cfg *config.Config) error {
	if options == nil {
		return nil
	}
	raw, ok := options["num_ctx"]
	if !ok {
		return nil
	}
	value, ok := exactIntegerOptionValue(raw)
	if !ok || value <= 0 {
		return fmt.Errorf("options.num_ctx must be a positive integer")
	}
	expected := cfg.OllamaCompatibility.ServerManagedNumCtx
	if expected == 0 {
		return fmt.Errorf("options.num_ctx is unsupported by the OpenAI backend; configure ollama_compatibility.server_managed_num_ctx to acknowledge the server-managed context window")
	}
	if value != expected {
		return fmt.Errorf("options.num_ctx=%d conflicts with server-managed value %d", value, expected)
	}
	delete(options, "num_ctx")
	return nil
}

func exactIntegerOptionValue(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		if math.Trunc(v) != v {
			return 0, false
		}
		return int(v), true
	case float32:
		if float32(math.Trunc(float64(v))) != v {
			return 0, false
		}
		return int(v), true
	default:
		return 0, false
	}
}
