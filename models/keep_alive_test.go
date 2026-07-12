package models

import (
	"encoding/json"
	"testing"
)

func TestOllamaRequestsAcceptKeepAliveStringsAndNumbers(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{name: "duration string", json: `"5m"`, want: "5m"},
		{name: "negative integer", json: `-1`, want: "-1"},
		{name: "negative float", json: `-1.0`, want: "-1"},
		{name: "zero", json: `0`, want: "0"},
		{name: "seconds", json: `3600`, want: "3600s"},
		{name: "fractional seconds", json: `1.5`, want: "1.5s"},
	}

	requestTypes := []struct {
		name      string
		unmarshal func(t *testing.T, body []byte) string
	}{
		{
			name: "chat",
			unmarshal: func(t *testing.T, body []byte) string {
				var req ChatRequest
				if err := json.Unmarshal(body, &req); err != nil {
					t.Fatalf("json.Unmarshal() error = %v", err)
				}
				return req.KeepAlive
			},
		},
		{
			name: "generate",
			unmarshal: func(t *testing.T, body []byte) string {
				var req GenerateRequest
				if err := json.Unmarshal(body, &req); err != nil {
					t.Fatalf("json.Unmarshal() error = %v", err)
				}
				return req.KeepAlive
			},
		},
	}

	for _, requestType := range requestTypes {
		for _, tt := range tests {
			t.Run(requestType.name+"/"+tt.name, func(t *testing.T) {
				body := []byte(`{"model":"test","keep_alive":` + tt.json + `}`)
				if got := requestType.unmarshal(t, body); got != tt.want {
					t.Fatalf("KeepAlive = %q, want %q", got, tt.want)
				}
			})
		}
	}
}

func TestOllamaRequestsRejectInvalidKeepAliveType(t *testing.T) {
	for _, target := range []any{&ChatRequest{}, &GenerateRequest{}} {
		err := json.Unmarshal([]byte(`{"model":"test","keep_alive":true}`), target)
		if err == nil {
			t.Fatalf("json.Unmarshal() accepted boolean keep_alive for %T", target)
		}
	}
}

func TestChatRequestNumericKeepAliveRegression(t *testing.T) {
	body := []byte(`{"model":"gemma4-31b","stream":true,"options":{"num_ctx":14000.0},"keep_alive":-1.0,"messages":[{"role":"user","content":"Hey are you there?"}]}`)

	var req ChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("json.Unmarshal() rejected request #5051 shape: %v", err)
	}
	if req.KeepAlive != "-1" {
		t.Fatalf("KeepAlive = %q, want %q", req.KeepAlive, "-1")
	}
}
