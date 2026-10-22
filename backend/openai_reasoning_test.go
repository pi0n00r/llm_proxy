package backend

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"llm_proxy/models"
)

func collectChat(t *testing.T, b *OpenAIBackend, stream bool) (content, thinking string) {
	t.Helper()
	respChan, _, err := b.Chat(context.Background(), models.ChatRequest{
		Model:    "gemma-4",
		Stream:   stream,
		Messages: []models.Message{{Role: "user", Content: "sum"}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	var c, th strings.Builder
	for resp := range respChan {
		c.WriteString(resp.Message.Content)
		th.WriteString(resp.Message.Thinking)
	}
	return c.String(), th.String()
}

func TestOpenAIBackendChatForwardsReasoning(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"choices":[{"delta":{"role":"assistant","reasoning":"Sum S3 "},"finish_reason":null}]}`,
		`data: {"choices":[{"delta":{"reasoning":"and S5."},"finish_reason":null}]}`,
		`data: {"choices":[{"delta":{"content":"2103"},"finish_reason":null}]}`,
		`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}`,
		`data: [DONE]`,
		"",
	}, "\n")
	nonStream := `{"choices":[{"index":0,"message":{"role":"assistant","content":"2103","reasoning":"Sum S3 and S5."},"finish_reason":"stop"}]}`

	tests := []struct {
		name      string
		stream    bool
		gemma4Fix bool
	}{
		{"non-streaming", false, false},
		{"streaming", true, false},
		{"streaming gemma4 fix", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewOpenAIBackend("http://backend.test", 10, false, tt.gemma4Fix)
			b.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if tt.stream {
					return textResponse("text/event-stream", sse), nil
				}
				return textResponse("application/json", nonStream), nil
			})

			content, thinking := collectChat(t, b, tt.stream)
			if content != "2103" {
				t.Fatalf("content = %q, want 2103", content)
			}
			if thinking != "Sum S3 and S5." {
				t.Fatalf("thinking = %q, want forwarded reasoning", thinking)
			}
		})
	}
}
