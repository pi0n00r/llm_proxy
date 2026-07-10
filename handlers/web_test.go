package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHomeHandlerDisplaysConfigSettings(t *testing.T) {
	data := map[string]interface{}{
		"BackendType":          "ollama",
		"BackendEndpoint":      "http://localhost:11434",
		"ServerHost":           "127.0.0.1",
		"ServerPort":           11435,
		"Timeout":              123,
		"DatabasePath":         "/tmp/llm_proxy.db",
		"EnableCORS":           true,
		"ToolBlacklist":        []string{"shell", "browser"},
		"PromptCacheEnabled":   false,
		"MaxTokensPolicy":      "drop_above",
		"MaxTokensLimit":       4096,
		"StreamOverrideMode":   "always",
		"OllamaThinkingMode":   "off",
		"Gemma4FixEnabled":     false,
		"TextInjectionEnabled": true,
		"TextInjectionText":    "extra instruction",
		"TextInjectionMode":    "system",
		"LogMessages":          true,
		"LogRawRequests":       true,
		"LogRawResponses":      false,
		"Verbose":              true,
		"DatabaseMaxRequests":  250,
		"DatabaseCleanupMins":  15,
	}

	rec := httptest.NewRecorder()
	NewWebHandler(nil, data).HomeHandler(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != 200 {
		t.Fatalf("HomeHandler status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Backend",
		"http://localhost:11434",
		"Stream Override",
		"always",
		"Ollama Thinking",
		"off",
		"Verbose Logs",
		"Message Logs",
		"Raw Request Logs",
		"Raw Response Logs",
		"Database",
		"/tmp/llm_proxy.db",
		"250",
		"15 min",
		"Max Tokens Policy",
		"drop_above",
		"limit: 4096",
		"Text Injection",
		"system",
		"Tool Blacklist",
		"shell, browser",
		"extra instruction",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("home page missing %q\nbody:\n%s", want, body)
		}
	}
}

func TestHomeHandlerHidesBackendSpecificSettings(t *testing.T) {
	data := map[string]interface{}{
		"BackendType":          "openai",
		"BackendEndpoint":      "http://localhost:8000/v1",
		"ServerHost":           "127.0.0.1",
		"ServerPort":           11435,
		"Timeout":              123,
		"DatabasePath":         "/tmp/llm_proxy.db",
		"EnableCORS":           false,
		"PromptCacheEnabled":   true,
		"StreamOverrideMode":   "passthrough",
		"OllamaThinkingMode":   "on",
		"Gemma4FixEnabled":     true,
		"TextInjectionEnabled": false,
		"DatabaseMaxRequests":  100,
		"DatabaseCleanupMins":  5,
	}

	rec := httptest.NewRecorder()
	NewWebHandler(nil, data).HomeHandler(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != 200 {
		t.Fatalf("HomeHandler status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "Ollama Thinking") {
		t.Fatalf("home page showed Ollama-only setting for OpenAI backend")
	}
	for _, want := range []string{"Force Prompt Cache", "Gemma 4 Fix"} {
		if !strings.Contains(body, want) {
			t.Fatalf("home page missing %q", want)
		}
	}
}
