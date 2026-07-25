package backend

import (
	"encoding/json"
	"testing"

	"llm_proxy/models"
)

func TestBuildOpenAIChatRequestTranslatesOllamaJSONFormatAndThinking(t *testing.T) {
	think := false
	b := NewOpenAIBackend("http://backend.test", 10, false, false)
	data, err := b.buildOpenAIChatRequest(models.ChatRequest{
		Model:    "gemma4-aimee",
		Messages: []models.Message{{Role: "user", Content: "return JSON"}},
		Format:   json.RawMessage(`"json"`),
		Think:    &think,
	}, []models.Message{{Role: "user", Content: "return JSON"}})
	if err != nil {
		t.Fatalf("buildOpenAIChatRequest() error = %v", err)
	}

	var got map[string]json.RawMessage
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if string(got["response_format"]) != `{"type":"json_object"}` {
		t.Fatalf("response_format = %s, want json_object", got["response_format"])
	}
	if string(got["reasoning_effort"]) != `"none"` {
		t.Fatalf("reasoning_effort = %s, want none", got["reasoning_effort"])
	}
	if _, ok := got["think"]; ok {
		t.Fatal("translated request retained Ollama think field")
	}
}

func TestOllamaResponseFormatWrapsSchemaNonStrict(t *testing.T) {
	got, err := ollamaResponseFormat(json.RawMessage(`{"type":"object","properties":{"answer":{"type":"string"}},"required":["answer"]}`))
	if err != nil {
		t.Fatalf("ollamaResponseFormat() error = %v", err)
	}

	var envelope struct {
		Type       string `json:"type"`
		JSONSchema struct {
			Name   string                 `json:"name"`
			Strict bool                   `json:"strict"`
			Schema map[string]interface{} `json:"schema"`
		} `json:"json_schema"`
	}
	if err := json.Unmarshal(got, &envelope); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if envelope.Type != "json_schema" || envelope.JSONSchema.Name != "ollama_response" {
		t.Fatalf("envelope identity = %#v", envelope)
	}
	if envelope.JSONSchema.Strict {
		t.Fatal("strict = true, want false for a plain Ollama schema")
	}
	if envelope.JSONSchema.Schema["type"] != "object" {
		t.Fatalf("schema = %#v, want original schema", envelope.JSONSchema.Schema)
	}
}

func TestOllamaResponseFormatPreservesExplicitJSONSchemaStrictness(t *testing.T) {
	input := json.RawMessage(`{"type":"json_schema","json_schema":{"name":"answer","strict":true,"schema":{"type":"object"}}}`)
	got, err := ollamaResponseFormat(input)
	if err != nil {
		t.Fatalf("ollamaResponseFormat() error = %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(got, &envelope); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	jsonSchema := envelope["json_schema"].(map[string]interface{})
	if strict, _ := jsonSchema["strict"].(bool); !strict {
		t.Fatalf("strict = %#v, want explicit true", jsonSchema["strict"])
	}
}

func TestBuildOpenAIChatRequestRemovesRawThink(t *testing.T) {
	think := false
	b := NewOpenAIBackend("http://backend.test", 10, false, false)
	data, err := b.buildOpenAIChatRequest(models.ChatRequest{
		Model:    "gemma4-aimee",
		Messages: []models.Message{{Role: "user", Content: "hello"}},
		Think:    &think,
		OpenAIRaw: map[string]json.RawMessage{
			"model":    json.RawMessage(`"gemma4-aimee"`),
			"messages": json.RawMessage(`[{"role":"user","content":"hello"}]`),
			"think":    json.RawMessage(`false`),
		},
	}, []models.Message{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("buildOpenAIChatRequest() error = %v", err)
	}

	var got map[string]json.RawMessage
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, ok := got["think"]; ok {
		t.Fatal("raw OpenAI request retained Ollama think field")
	}
	if string(got["reasoning_effort"]) != `"none"` {
		t.Fatalf("reasoning_effort = %s, want none", got["reasoning_effort"])
	}
}
