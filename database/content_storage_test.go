package database

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLogWithoutContentStorageRetainsOnlyMetadata(t *testing.T) {
	db, err := NewWithContentStorage(filepath.Join(t.TempDir(), "llm_proxy.db"), false)
	if err != nil {
		t.Fatalf("NewWithContentStorage() error = %v", err)
	}
	defer db.Close()

	entry := LogEntry{
		Timestamp:        time.Now(),
		Endpoint:         "/api/chat",
		Method:           "POST",
		Model:            "gemma4-aimee",
		Prompt:           "private prompt",
		Response:         "private response",
		StatusCode:       500,
		LatencyMs:        42,
		BackendType:      "openai",
		Error:            "backend body containing private response",
		FrontendRequest:  "private frontend request",
		FrontendResponse: "private frontend response",
		BackendRequest:   "private backend request",
		BackendResponse:  "private backend response",
		LastMessage:      "private last message",
	}
	if err := db.Log(entry); err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	entries, err := db.GetRecentEntries(1, 0)
	if err != nil {
		t.Fatalf("GetRecentEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	got := entries[0]
	if got.Endpoint != entry.Endpoint || got.Model != entry.Model || got.StatusCode != entry.StatusCode || got.LatencyMs != entry.LatencyMs {
		t.Fatalf("metadata changed: %#v", got)
	}
	if got.Prompt != "" || got.Response != "" || got.FrontendRequest != "" ||
		got.FrontendResponse != "" || got.BackendRequest != "" ||
		got.BackendResponse != "" || got.LastMessage != "" {
		t.Fatalf("content was retained: %#v", got)
	}
	if got.Error != "request failed; content storage disabled" {
		t.Fatalf("Error = %q, want content-free failure marker", got.Error)
	}
}
