package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"llm_proxy/config"
	"llm_proxy/database"
	"llm_proxy/models"
)

func TestOllamaThinkingOverrideForcesChatThink(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		mode     string
		want     bool
	}{
		{name: "openai chat off", endpoint: "openai_chat", mode: "off", want: false},
		{name: "openai chat on", endpoint: "openai_chat", mode: "on", want: true},
		{name: "ollama chat off", endpoint: "ollama_chat", mode: "off", want: false},
		{name: "ollama chat on", endpoint: "ollama_chat", mode: "on", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spy, db, cfg := newStreamOverrideTest(t)
			cfg.OllamaThinkingOverride.Mode = tt.mode

			rec := serveThinkingOverrideRequest(t, tt.endpoint, spy, db, cfg, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
			if spy.lastChatReq.Think == nil {
				t.Fatal("backend think = nil, want non-nil")
			}
			if *spy.lastChatReq.Think != tt.want {
				t.Fatalf("backend think = %v, want %v", *spy.lastChatReq.Think, tt.want)
			}
		})
	}
}

func TestOllamaThinkingOverridePassthrough(t *testing.T) {
	for _, endpoint := range []string{"openai_chat", "ollama_chat"} {
		t.Run(endpoint+" leaves absent think nil", func(t *testing.T) {
			spy, db, cfg := newStreamOverrideTest(t)
			cfg.OllamaThinkingOverride.Mode = "passthrough"

			rec := serveThinkingOverrideRequest(t, endpoint, spy, db, cfg, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
			if spy.lastChatReq.Think != nil {
				t.Fatalf("backend think = %v, want nil", *spy.lastChatReq.Think)
			}
		})

		t.Run(endpoint+" preserves client sent think", func(t *testing.T) {
			spy, db, cfg := newStreamOverrideTest(t)
			cfg.OllamaThinkingOverride.Mode = "passthrough"
			clientThink := true

			rec := serveThinkingOverrideRequest(t, endpoint, spy, db, cfg, &clientThink)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
			if spy.lastChatReq.Think == nil {
				t.Fatal("backend think = nil, want client value")
			}
			if *spy.lastChatReq.Think != clientThink {
				t.Fatalf("backend think = %v, want %v", *spy.lastChatReq.Think, clientThink)
			}
		})
	}
}

func serveThinkingOverrideRequest(t *testing.T, endpoint string, spy *streamOverrideSpyBackend, db *database.DB, cfg *config.Config, think *bool) *httptest.ResponseRecorder {
	t.Helper()

	var (
		handler http.Handler
		path    string
		body    string
	)

	switch endpoint {
	case "openai_chat":
		handler = NewOpenAIChatCompletionsHandler(spy, db, cfg)
		path = "/v1/chat/completions"
		reqBody := map[string]interface{}{
			"model":    "test-model",
			"messages": []models.Message{{Role: "user", Content: "hello"}},
		}
		if think != nil {
			reqBody["think"] = *think
		}
		body = marshalThinkingOverrideBody(t, reqBody)
	case "ollama_chat":
		handler = NewChatHandler(spy, db, cfg)
		path = "/api/chat"
		body = marshalThinkingOverrideBody(t, models.ChatRequest{
			Model:    "test-model",
			Messages: []models.Message{{Role: "user", Content: "hello"}},
			Think:    think,
		})
	default:
		t.Fatalf("unknown endpoint %q", endpoint)
	}

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func marshalThinkingOverrideBody(t *testing.T, body interface{}) string {
	t.Helper()

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return string(data)
}
