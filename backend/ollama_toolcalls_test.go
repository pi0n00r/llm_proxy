package backend

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"llm_proxy/models"
)

func TestNormalizeToolCallArgumentsToObjects(t *testing.T) {
	originalArgs := `{"text":"x"}`
	objectArgs := map[string]interface{}{"already": "object"}
	invalidArgs := `{"missing":`

	input := []models.Message{
		{
			Role: "assistant",
			ToolCalls: []interface{}{
				map[string]interface{}{
					"id": "call-valid",
					"function": map[string]interface{}{
						"name":      "lookup",
						"arguments": originalArgs,
					},
				},
				map[string]interface{}{
					"id": "call-object",
					"function": map[string]interface{}{
						"name":      "native",
						"arguments": objectArgs,
					},
				},
				map[string]interface{}{
					"id": "call-invalid",
					"function": map[string]interface{}{
						"name":      "broken",
						"arguments": invalidArgs,
					},
				},
			},
		},
		{Role: "tool", Content: `{"result":true}`},
	}

	got := normalizeToolCallArgumentsToObjects(input)

	validCall := got[0].ToolCalls[0].(map[string]interface{})
	validFn := validCall["function"].(map[string]interface{})
	validArgs, ok := validFn["arguments"].(map[string]interface{})
	if !ok {
		t.Fatalf("valid arguments type = %T, want map", validFn["arguments"])
	}
	if validArgs["text"] != "x" {
		t.Fatalf("valid arguments = %#v, want text x", validArgs)
	}

	objectCall := got[0].ToolCalls[1].(map[string]interface{})
	objectFn := objectCall["function"].(map[string]interface{})
	if !reflect.DeepEqual(objectFn["arguments"], objectArgs) {
		t.Fatalf("object arguments = %#v, want original object", objectFn["arguments"])
	}

	invalidCall := got[0].ToolCalls[2].(map[string]interface{})
	invalidFn := invalidCall["function"].(map[string]interface{})
	if invalidFn["arguments"] != invalidArgs {
		t.Fatalf("invalid arguments = %#v, want original string", invalidFn["arguments"])
	}

	inputCall := input[0].ToolCalls[0].(map[string]interface{})
	inputFn := inputCall["function"].(map[string]interface{})
	if inputFn["arguments"] != originalArgs {
		t.Fatalf("input was mutated: arguments = %#v, want original string", inputFn["arguments"])
	}
	if got[1].Content != input[1].Content {
		t.Fatalf("tool result content = %q, want %q", got[1].Content, input[1].Content)
	}
}

func TestOllamaBackendChatConvertsToolCallArgumentStringsInRawRequest(t *testing.T) {
	var gotReq models.ChatRequest

	b := NewOllamaBackend("http://backend.test", 10)
	b.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("path = %q, want /api/chat", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		return jsonResponse(`{"model":"test-model","message":{"role":"assistant","content":"ok"},"done":true}`), nil
	})

	respChan, meta, err := b.Chat(context.Background(), models.ChatRequest{
		Model: "test-model",
		Messages: []models.Message{
			{Role: "user", Content: "start"},
			{
				Role: "assistant",
				ToolCalls: []interface{}{
					map[string]interface{}{
						"id":   "call-empty",
						"type": "function",
						"function": map[string]interface{}{
							"name":      "get_world_context",
							"arguments": "{}",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	for range respChan {
	}

	gotCall := gotReq.Messages[1].ToolCalls[0].(map[string]interface{})
	gotFn := gotCall["function"].(map[string]interface{})
	gotArgs, ok := gotFn["arguments"].(map[string]interface{})
	if !ok {
		t.Fatalf("backend request arguments type = %T, want map; raw request = %s", gotFn["arguments"], meta.RawRequest)
	}
	if len(gotArgs) != 0 {
		t.Fatalf("backend request arguments = %#v, want empty object", gotArgs)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(meta.RawRequest), &raw); err != nil {
		t.Fatalf("RawRequest is not JSON: %v", err)
	}
	rawMessages := raw["messages"].([]interface{})
	rawAssistant := rawMessages[1].(map[string]interface{})
	rawToolCalls := rawAssistant["tool_calls"].([]interface{})
	rawCall := rawToolCalls[0].(map[string]interface{})
	rawFn := rawCall["function"].(map[string]interface{})
	if _, ok := rawFn["arguments"].(map[string]interface{}); !ok {
		t.Fatalf("raw request arguments type = %T, want map; raw request = %s", rawFn["arguments"], meta.RawRequest)
	}
}
