package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"llm_proxy/backend"
	"llm_proxy/config"
	"llm_proxy/database"
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

type compatibilitySpyBackend struct {
	lastChatReq     models.ChatRequest
	lastGenerateReq models.GenerateRequest
}

func (s *compatibilitySpyBackend) Generate(_ context.Context, req models.GenerateRequest) (<-chan models.GenerateResponse, *backend.BackendMetadata, error) {
	s.lastGenerateReq = req
	ch := make(chan models.GenerateResponse, 1)
	ch <- models.GenerateResponse{Model: req.Model, CreatedAt: time.Now(), Done: true}
	close(ch)
	return ch, &backend.BackendMetadata{}, nil
}

func (s *compatibilitySpyBackend) Chat(_ context.Context, req models.ChatRequest) (<-chan models.ChatResponse, *backend.BackendMetadata, error) {
	s.lastChatReq = req
	ch := make(chan models.ChatResponse, 1)
	ch <- models.ChatResponse{Model: req.Model, CreatedAt: time.Now(), Done: true}
	close(ch)
	return ch, &backend.BackendMetadata{}, nil
}

func (s *compatibilitySpyBackend) ListModels(context.Context) (models.ModelsResponse, error) {
	return models.ModelsResponse{}, nil
}

func (s *compatibilitySpyBackend) ShowModel(context.Context, string) (models.ShowResponse, error) {
	return models.ShowResponse{}, nil
}

func TestServerManagedCompatibilityHTTPPaths(t *testing.T) {
	endpoints := []struct {
		name string
		path string
		body func(keepAlive string, numCtx string) string
	}{
		{
			name: "chat",
			path: "/api/chat",
			body: func(keepAlive string, numCtx string) string {
				return `{"model":"test","messages":[{"role":"user","content":"hello"}],"keep_alive":` + keepAlive + `,"options":{"num_ctx":` + numCtx + `}}`
			},
		},
		{
			name: "generate",
			path: "/api/generate",
			body: func(keepAlive string, numCtx string) string {
				return `{"model":"test","prompt":"hello","keep_alive":` + keepAlive + `,"options":{"num_ctx":` + numCtx + `}}`
			},
		},
	}
	cases := []struct {
		name      string
		keepAlive string
		numCtx    string
		wantCode  int
	}{
		{name: "Home Assistant 2026.6.4 duration", keepAlive: `"-1s"`, numCtx: "131072", wantCode: http.StatusOK},
		{name: "numeric negative one", keepAlive: `-1`, numCtx: "131072", wantCode: http.StatusOK},
		{name: "negative number string", keepAlive: `"-2"`, numCtx: "131072", wantCode: http.StatusOK},
		{name: "negative duration", keepAlive: `"-5m"`, numCtx: "131072", wantCode: http.StatusOK},
		{name: "finite keep alive conflict", keepAlive: `"5m"`, numCtx: "131072", wantCode: http.StatusBadRequest},
		{name: "context conflict", keepAlive: `"-1s"`, numCtx: "32768", wantCode: http.StatusBadRequest},
	}

	for _, endpoint := range endpoints {
		for _, tt := range cases {
			t.Run(endpoint.name+"/"+tt.name, func(t *testing.T) {
				spy, db, cfg := newCompatibilityHTTPTest(t)
				var handler http.Handler
				if endpoint.path == "/api/chat" {
					handler = NewChatHandler(spy, db, cfg)
				} else {
					handler = NewGenerateHandler(spy, db, cfg)
				}

				req := httptest.NewRequest(http.MethodPost, endpoint.path, strings.NewReader(endpoint.body(tt.keepAlive, tt.numCtx)))
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				if rec.Code != tt.wantCode {
					t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.wantCode, rec.Body.String())
				}
				if tt.wantCode == http.StatusBadRequest && !strings.Contains(rec.Body.String(), "conflicts with server-managed value") {
					t.Fatalf("HTTP 400 body = %q, want explicit conflict response", rec.Body.String())
				}
				if tt.wantCode != http.StatusOK {
					return
				}

				if endpoint.path == "/api/chat" {
					if spy.lastChatReq.KeepAlive != "" {
						t.Fatalf("forwarded keep_alive = %q, want removed", spy.lastChatReq.KeepAlive)
					}
					if _, ok := spy.lastChatReq.Options["num_ctx"]; ok {
						t.Fatal("forwarded chat options retained num_ctx")
					}
				} else {
					if spy.lastGenerateReq.KeepAlive != "" {
						t.Fatalf("forwarded keep_alive = %q, want removed", spy.lastGenerateReq.KeepAlive)
					}
					if _, ok := spy.lastGenerateReq.Options["num_ctx"]; ok {
						t.Fatal("forwarded generate options retained num_ctx")
					}
				}
			})
		}
	}
}

func newCompatibilityHTTPTest(t *testing.T) (*compatibilitySpyBackend, *database.DB, *config.Config) {
	t.Helper()
	db, err := database.New(filepath.Join(t.TempDir(), "llm_proxy.db"))
	if err != nil {
		t.Fatalf("database.New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close() error = %v", err)
		}
	})
	return &compatibilitySpyBackend{}, db, fleetCompatibilityConfig()
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
