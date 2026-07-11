package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"llm_proxy/database"
)

func TestHomeHandlerDisplaysConfigSettings(t *testing.T) {
	data := map[string]interface{}{
		"BackendType":          "ollama",
		"BackendEndpoint":      "http://localhost:11434",
		"ServerHost":           "127.0.0.1",
		"ServerPort":           11435,
		"Timeout":              123,
		"DatabasePath":         "/tmp/llm_proxy.db",
		"EnableCORS":           true,
		"ToolBlacklist":        []string{"shell", "browser"},
		"PromptCacheEnabled":   false,
		"MaxTokensPolicy":      "drop_above",
		"MaxTokensLimit":       4096,
		"StreamOverrideMode":   "always",
		"OllamaThinkingMode":   "off",
		"Gemma4FixEnabled":     false,
		"TextInjectionEnabled": true,
		"TextInjectionText":    "extra instruction",
		"TextInjectionMode":    "system",
		"LogMessages":          true,
		"LogRawRequests":       true,
		"LogRawResponses":      false,
		"Verbose":              true,
		"DatabaseMaxRequests":  250,
		"DatabaseCleanupMins":  15,
	}

	rec := httptest.NewRecorder()
	NewWebHandler(nil, data).HomeHandler(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != 200 {
		t.Fatalf("HomeHandler status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Backend",
		"http://localhost:11434",
		"Stream Override",
		"always",
		"Ollama Thinking",
		"off",
		"Verbose Logs",
		"Message Logs",
		"Raw Request Logs",
		"Raw Response Logs",
		"Database",
		"/tmp/llm_proxy.db",
		"250",
		"15 min",
		"Max Tokens Policy",
		"drop_above",
		"limit: 4096",
		"Text Injection",
		"system",
		"Tool Blacklist",
		"shell, browser",
		"extra instruction",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("home page missing %q\nbody:\n%s", want, body)
		}
	}
}

func TestStatusOK(t *testing.T) {
	for _, status := range []int{200, 201, 204, 299} {
		if !statusOK(status) {
			t.Errorf("statusOK(%d) = false", status)
		}
	}
	for _, status := range []int{0, 199, 300, 400, 500} {
		if statusOK(status) {
			t.Errorf("statusOK(%d) = true", status)
		}
	}
}

func TestRenderedMessagesIncludeReadableToolCalls(t *testing.T) {
	raw := `{"messages":[{"role":"assistant","content":"","tool_calls":[{"id":"call-1","function":{"name":"turn_on","arguments":"{\"room\":\"office\"}"}}]},{"role":"tool","tool_call_id":"call-1","content":"done"}]}`
	messages := renderedMessagesFromRaw(raw)
	if len(messages) != 2 || len(messages[0].ToolCalls) != 1 {
		t.Fatalf("rendered messages = %#v", messages)
	}
	call := messages[0].ToolCalls[0]
	if call.Name != "turn_on" || call.ID != "call-1" || !strings.Contains(call.Arguments, "\n  \"room\"") {
		t.Fatalf("rendered tool call = %#v", call)
	}
	if messages[1].ToolCallID != "call-1" {
		t.Fatalf("tool result ID = %q", messages[1].ToolCallID)
	}
}

func TestRenderedToolResultPrettyPrintsJSON(t *testing.T) {
	raw := `{"messages":[{"role":"tool","content":"{\"ok\":true,\"items\":[1,2]}"},{"role":"tool","content":"{not json}"}]}`
	messages := renderedMessagesFromRaw(raw)
	if len(messages) != 2 {
		t.Fatalf("rendered messages = %#v", messages)
	}
	if got := messages[0].Parts[0]; got.Kind != "json" || !strings.Contains(got.Text, "\n  \"ok\": true") {
		t.Fatalf("formatted JSON result = %#v", got)
	}
	if got := messages[1].Parts[0]; got.Kind != "text" || got.Text != "{not json}" {
		t.Fatalf("invalid JSON result = %#v", got)
	}
}

func TestConversationChangesAreMarked(t *testing.T) {
	frontend := []renderedLogMessage{{Role: "system", Summary: "original"}, {Role: "user", Summary: "hello"}}
	backend := []renderedLogMessage{{Role: "system", Summary: "injected"}, {Role: "user", Summary: "hello"}, {Role: "system", Summary: "added"}}
	markConversationChanges(frontend, backend)
	if !backend[0].Changed || backend[1].Changed || !backend[2].Changed {
		t.Fatalf("unexpected changed flags: %#v", backend)
	}
}

func TestDetailsTemplateHasConversationTabsAndAPIHandoff(t *testing.T) {
	data := struct {
		*database.LogEntry
		NextID               *int64
		PrevID               *int64
		PromptDisplay        string
		FrontendConversation []renderedLogMessage
		BackendConversation  []renderedLogMessage
	}{
		LogEntry:             &database.LogEntry{ID: 42, StatusCode: 200, Response: "final answer"},
		FrontendConversation: []renderedLogMessage{{Role: "user", Parts: []renderedLogPart{{Kind: "text", Text: "hello"}}}},
		BackendConversation:  []renderedLogMessage{{Role: "assistant", ToolCalls: []renderedToolCall{{Name: "search", Arguments: "{}"}}, Changed: true}},
	}
	var out strings.Builder
	if err := templates.ExecuteTemplate(&out, "details.html", data); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Client → Proxy", "Proxy → Backend", "CHANGED BY PROXY", "Tool call", "search", "MODEL RESPONSE", "final answer", "/api/logs/42", "Copy for agent"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("details page missing %q", want)
		}
	}
}

func TestHomeHandlerHidesBackendSpecificSettings(t *testing.T) {
	data := map[string]interface{}{
		"BackendType":          "openai",
		"BackendEndpoint":      "http://localhost:8000/v1",
		"ServerHost":           "127.0.0.1",
		"ServerPort":           11435,
		"Timeout":              123,
		"DatabasePath":         "/tmp/llm_proxy.db",
		"EnableCORS":           false,
		"PromptCacheEnabled":   true,
		"StreamOverrideMode":   "passthrough",
		"OllamaThinkingMode":   "on",
		"Gemma4FixEnabled":     true,
		"TextInjectionEnabled": false,
		"DatabaseMaxRequests":  100,
		"DatabaseCleanupMins":  5,
	}

	rec := httptest.NewRecorder()
	NewWebHandler(nil, data).HomeHandler(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != 200 {
		t.Fatalf("HomeHandler status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "Ollama Thinking") {
		t.Fatalf("home page showed Ollama-only setting for OpenAI backend")
	}
	for _, want := range []string{"Force Prompt Cache", "Gemma 4 Fix"} {
		if !strings.Contains(body, want) {
			t.Fatalf("home page missing %q", want)
		}
	}
}
