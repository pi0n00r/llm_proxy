package models

import (
	"encoding/json"
	"testing"
)

func TestGenerateRequestDecodesThinkFalse(t *testing.T) {
	var req GenerateRequest
	if err := json.Unmarshal([]byte(`{"model":"gemma4-aimee","prompt":"hello","think":false}`), &req); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if req.Think == nil || *req.Think {
		t.Fatalf("Think = %#v, want pointer to false", req.Think)
	}
}
