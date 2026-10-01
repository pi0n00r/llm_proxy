package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestModelAliasConfigRequiresCompleteOpenAIOnlyPair(t *testing.T) {
	tests := []struct {
		name    string
		backend string
		fields  string
		valid   bool
	}{
		{name: "valid", backend: "openai", fields: "model_alias = \"gemma4\"\nupstream_model = \"physical-gemma\"\n", valid: true},
		{name: "alias only", backend: "openai", fields: "model_alias = \"gemma4\"\n"},
		{name: "upstream only", backend: "openai", fields: "upstream_model = \"physical-gemma\"\n"},
		{name: "wrong backend", backend: "ollama", fields: "model_alias = \"gemma4\"\nupstream_model = \"physical-gemma\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			body := "[backend]\ntype = \"" + tt.backend + "\"\n" + tt.fields
			if err := os.WriteFile(path, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if tt.valid && (err != nil || cfg.Backend.ModelAlias != "gemma4") {
				t.Fatalf("valid config: cfg=%+v err=%v", cfg, err)
			}
			if !tt.valid && err == nil {
				t.Fatalf("invalid config accepted: %+v", cfg)
			}
		})
	}
}
