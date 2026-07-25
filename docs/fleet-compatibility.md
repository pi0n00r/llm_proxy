# Fleet Compatibility Contract

This fork adds narrow Ollama-to-OpenAI translations without changing the
native Ollama backend.

## Structured Output

For OpenAI-compatible backends:

- `format: "json"` becomes `response_format: {"type":"json_object"}`.
- A plain Ollama JSON Schema becomes:

  ```json
  {
    "type": "json_schema",
    "json_schema": {
      "name": "ollama_response",
      "strict": false,
      "schema": {}
    }
  }
  ```

- An explicit OpenAI-style `json_schema` envelope is preserved, including an
  explicitly requested `strict: true`.
- Unsupported format strings are rejected rather than ignored.

## Request-Scoped Reasoning

Client-supplied `think: false` becomes `reasoning_effort: "none"` for
OpenAI-backed `/api/chat` and `/api/generate`. It does not change model
configuration or another client's requests. `think: true` leaves the backend
default in place.

## Server-Managed Ollama Hints

OpenAI Chat Completions has no request fields equivalent to Ollama's model
residence and context allocation controls. Configure the values already
enforced by the server:

```toml
[ollama_compatibility]
server_managed_keep_alive = "-1"
server_managed_num_ctx = 131072
```

A matching `keep_alive` or `options.num_ctx` is accepted and removed before
forwarding. Ollama defines any negative keepalive number or duration as
indefinite, so numeric `-1`, string `"-1"`, and negative duration strings such
as `"-1s"` are equivalent when the configured server-managed keepalive is
indefinite. Finite or malformed keepalive claims and non-matching context
values return HTTP 400. If the corresponding server-managed value is not
configured, sending that hint also returns HTTP 400. The proxy never reports
that it honored a value it cannot apply.

## Generate Translation

For an OpenAI-compatible backend, `/api/generate` is expressed through
`/v1/chat/completions`. The optional system text becomes a system message and
the prompt becomes a user message. Streaming, supported generation options,
structured output, and request-scoped `think: false` use the same path as
`/api/chat`.

## Tool Calls

Both streamed and non-streamed Ollama responses expose `function.arguments`
as a JSON object. Tool-call IDs are retained so a subsequent `role: "tool"`
message can carry the matching `tool_call_id`. OpenAI-only `type` and `index`
members are not leaked into non-streamed Ollama tool-call objects.

## Content Storage

Set:

```toml
[database]
store_content = false
```

SQLite retains request metadata but stores no prompts, responses, raw
frontend/backend bodies, tool results, last-message previews, or detailed
error bodies. Disable the three stdout content logging options separately for
a content-free runtime configuration.

## Listener

Addresses are built with `net.JoinHostPort`. `host = "::"` therefore becomes
`[::]:<port>` and binds dual-stack on operating systems configured to accept
IPv4-mapped IPv6 connections.

## Deliberate Omission

`/api/embed` is not implemented. It should be added only with an identified
consumer and a corresponding compatibility test.
