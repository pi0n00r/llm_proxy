package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOpenAIChatChoiceUnmarshalMapsReasoningFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"vllm reasoning", `{"choices":[{"message":{"role":"assistant","content":"2103","reasoning":"think"}}]}`},
		{"reasoning_content", `{"choices":[{"message":{"role":"assistant","content":"2103","reasoning_content":"think"}}]}`},
		{"ollama thinking", `{"choices":[{"message":{"role":"assistant","content":"2103","thinking":"think"}}]}`},
		{"delta reasoning", `{"choices":[{"delta":{"reasoning":"think"}}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp OpenAIChatResponse
			if err := json.Unmarshal([]byte(tt.body), &resp); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			msg := resp.Choices[0].Message
			if msg == nil {
				msg = resp.Choices[0].Delta
			}
			if msg.Thinking != "think" {
				t.Fatalf("Thinking = %q, want think", msg.Thinking)
			}
		})
	}
}

func TestOpenAIChatChoiceMarshalEmitsOpenAIReasoningFields(t *testing.T) {
	resp := OpenAIChatResponse{
		Choices: []OpenAIChatChoice{{
			Message:      &Message{Role: "assistant", Content: "2103", Thinking: "think"},
			FinishReason: "stop",
		}},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	got := string(data)
	for _, want := range []string{`"reasoning_content":"think"`, `"content":"2103"`, `"finish_reason":"stop"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("json = %s, want %s", got, want)
		}
	}
	if strings.Contains(got, `"thinking"`) || strings.Contains(got, `"reasoning"`) || strings.Contains(got, `"delta"`) {
		t.Fatalf("json = %s, want no thinking, reasoning or delta keys", got)
	}
}

func TestMessageMarshalKeepsOllamaThinkingField(t *testing.T) {
	data, err := json.Marshal(Message{Role: "assistant", Content: "x", Thinking: "think"})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if !strings.Contains(string(data), `"thinking":"think"`) || strings.Contains(string(data), "reasoning") {
		t.Fatalf("json = %s, want Ollama thinking field only", data)
	}
}
