package handlers

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"llm_proxy/backend"
	"llm_proxy/config"
	"llm_proxy/database"
	"llm_proxy/models"
)

type requestBodySpyBackend struct {
	calls int
}

func (b *requestBodySpyBackend) Generate(context.Context, models.GenerateRequest) (<-chan models.GenerateResponse, *backend.BackendMetadata, error) {
	b.calls++
	responses := make(chan models.GenerateResponse)
	close(responses)
	return responses, &backend.BackendMetadata{}, nil
}

func (b *requestBodySpyBackend) Chat(context.Context, models.ChatRequest) (<-chan models.ChatResponse, *backend.BackendMetadata, error) {
	b.calls++
	responses := make(chan models.ChatResponse)
	close(responses)
	return responses, &backend.BackendMetadata{}, nil
}

func (b *requestBodySpyBackend) ListModels(context.Context) (models.ModelsResponse, error) {
	return models.ModelsResponse{}, nil
}

func (b *requestBodySpyBackend) ShowModel(context.Context, string) (models.ShowResponse, error) {
	return models.ShowResponse{}, nil
}

func TestCompletionHandlersRejectOversizedBodies(t *testing.T) {
	for _, endpoint := range []string{"/api/chat", "/api/generate", "/v1/chat/completions"} {
		t.Run(endpoint, func(t *testing.T) {
			handler, spy, db := newRequestBodyTestHandler(t, endpoint, 64)
			defer db.Close()

			req := httptest.NewRequest(http.MethodPost, endpoint, strings.NewReader(strings.Repeat("x", 65)))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusRequestEntityTooLarge, rec.Body.String())
			}
			if spy.calls != 0 {
				t.Fatalf("backend calls = %d, want 0", spy.calls)
			}

			entries, err := db.GetRecentEntries(1, 0)
			if err != nil {
				t.Fatalf("GetRecentEntries() error = %v", err)
			}
			if len(entries) != 1 || entries[0].StatusCode != http.StatusRequestEntityTooLarge {
				t.Fatalf("logged entries = %#v, want one 413 entry", entries)
			}
		})
	}
}

func TestMalformedBodiesAreNotLoggedUnlessRawLoggingIsEnabled(t *testing.T) {
	const sentinel = "request-body-secret"

	previousOutput := log.Writer()
	previousFlags := log.Flags()
	previousPrefix := log.Prefix()
	var logs bytes.Buffer
	log.SetOutput(&logs)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(previousOutput)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
	})

	for _, endpoint := range []string{"/api/chat", "/api/generate", "/v1/chat/completions"} {
		handler, spy, db := newRequestBodyTestHandler(t, endpoint, 1024)
		req := httptest.NewRequest(http.MethodPost, endpoint, strings.NewReader(`{"model":"`+sentinel+`"`))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		db.Close()

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d, want %d", endpoint, rec.Code, http.StatusBadRequest)
		}
		if spy.calls != 0 {
			t.Fatalf("%s backend calls = %d, want 0", endpoint, spy.calls)
		}
	}

	if strings.Contains(logs.String(), sentinel) {
		t.Fatalf("malformed request content leaked to stdout logs: %s", logs.String())
	}
}

func newRequestBodyTestHandler(t *testing.T, endpoint string, limit int64) (http.Handler, *requestBodySpyBackend, *database.DB) {
	t.Helper()
	db, err := database.NewWithContentStorage(filepath.Join(t.TempDir(), "llm_proxy.db"), false)
	if err != nil {
		t.Fatalf("NewWithContentStorage() error = %v", err)
	}
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:                "127.0.0.1",
			Port:                11434,
			MaxRequestBodyBytes: limit,
		},
		Backend:  config.BackendConfig{Type: "openai"},
		Database: config.DatabaseConfig{StoreContent: false},
	}
	spy := &requestBodySpyBackend{}

	switch endpoint {
	case "/api/chat":
		return NewChatHandler(spy, db, cfg), spy, db
	case "/api/generate":
		return NewGenerateHandler(spy, db, cfg), spy, db
	case "/v1/chat/completions":
		return NewOpenAIChatCompletionsHandler(spy, db, cfg), spy, db
	default:
		t.Fatalf("unsupported test endpoint %q", endpoint)
		return nil, nil, nil
	}
}
