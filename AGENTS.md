# AGENTS.md

## Project Overview

`llm_proxy` is a Go proxy server that exposes Ollama-compatible endpoints and forwards requests to either an OpenAI-compatible backend or an Ollama backend. It also exposes basic OpenAI-compatible frontend endpoints for simple clients.

The server logs requests and responses to SQLite and serves a small web UI for browsing logs. The repository also includes a dependency-free terminal chat client in `cmd/chatclient`.

## Important Commands

- Run all tests:
  ```bash
  CGO_ENABLED=0 go test ./...
  ```
- Run the proxy from source:
  ```bash
  ./run.sh
  ```
- Run the chat client from source:
  ```bash
  ./client.sh
  ```
- Run the chat client against OpenAI-compatible frontend endpoints:
  ```bash
  ./client.sh --openai
  ```
- Build the proxy binary:
  ```bash
  CGO_ENABLED=0 go build -o llm_proxy .
  ```

If the Go build cache is not writable in a sandbox, set it explicitly:

```bash
GOCACHE=/tmp/llm_proxy_go_build CGO_ENABLED=0 go test ./...
```

## Repository Layout

- `main.go` wires config, database, backend selection, HTTP routes, middleware, and shutdown.
- `config/` loads and validates `config.toml`.
- `backend/` contains the backend interface and the OpenAI/Ollama backend implementations.
- `handlers/` contains the Ollama frontend handlers, OpenAI frontend handlers, web UI handlers, templates, and static assets.
- `handlers/chat_features.go`, `request_sanitization.go`, and `stream_override.go` contain shared request-mutation behavior used across frontend endpoints.
- `backend/gemma4_fix.go` contains the optional, self-contained Gemma 4/vLLM streaming-corruption mitigation.
- `models/` contains shared request and response structs.
- `database/` owns SQLite initialization and log queries.
- `middleware/` contains CORS and verbose request logging middleware.
- `cmd/chatclient/` contains the stdlib-only terminal chat client.
- `run.sh` and `client.sh` are convenience wrappers for local manual testing.
- `docs/` contains the JSON logs API reference and design notes for the Gemma 4 mitigation.

## Streaming Architecture

- `backend.Backend.Chat`/`Generate` always return a `<-chan` of response chunks terminated by one item with `Done: true`, regardless of whether the upstream backend itself streamed. Frontend handlers (`handlers/chat.go`, `handlers/generate.go`, `handlers/openai_frontend.go`) just range over the channel.
- The *backend-facing* stream flag (does the proxy ask the upstream API for SSE) and the *client-facing* response format (does the proxy send the client SSE/NDJSON chunks or one JSON blob) are deliberately decoupled — `stream_override` only ever changes the former. **The client must always get back the format it originally requested**, regardless of `stream_override.mode`, because clients are coded against the format they asked for (e.g. an OpenAI SDK client that sent `stream:true` will error like "Stream ended without finish_reason" if it gets a plain JSON body instead).
- All three handlers capture `clientWantsStream := req.Stream` *before* calling `resolveStream`, then use `clientWantsStream` (not the overridden value) to decide what to send the client, while the overridden value goes to `backend.Chat`/`Generate`:
  - `handlers/openai_frontend.go`: `clientWantsStream` picks `streamResponse` (SSE) vs `writeResponse` (single JSON); both already aggregate/forward whatever the channel yields, so they work no matter how many chunks the backend actually produced.
  - `handlers/chat.go` / `handlers/generate.go`: the response-writing loop always builds a `combined` aggregate alongside writing, then only flushes per-chunk if `clientWantsStream`; if not, it sends the aggregate once at the end. This reshapes N backend chunks into 1 client response, or 1 backend chunk into a valid 1-chunk client stream, as needed.
- When adding new stream-related behavior, never let `stream_override` change `clientWantsStream` — only the value passed to the backend call.

## Adding a Config Option

Follow the existing pattern (see `RequestSanitizationConfig`, `StreamOverrideConfig` in `config/config.go`):
1. Add the field to a `...Config` struct with a `toml:"..."` tag.
2. If it's an enum-like string, validate allowed values in `Load()` and set its default there too (empty string from an unset TOML key is not validated against the enum).
3. Mirror the new section/key in `config.toml.example` with a comment listing allowed values.
4. Document it in `README.md` under "Configuration Options" (`config.toml` itself is gitignored — local/example only).
5. Add `config_test.go` cases for: explicit value loads, default-when-omitted, and rejection of an invalid value.
6. If it changes request handling, add a handler-level parity test across the three endpoints (`openai_chat`, `ollama_chat`, `ollama_generate`) using a spy backend — see `handlers/request_sanitization_test.go` or `handlers/stream_override_test.go` for the pattern.

## Development Notes

- Keep the chat client dependency-free. It should remain trivial to run with `go run ./cmd/chatclient`.
- The SQLite driver is `modernc.org/sqlite`, a pure-Go driver. Do not reintroduce `github.com/mattn/go-sqlite3` or CGO requirements unless explicitly requested.
- Use `database/sql` and the existing `database.DB` wrapper for database work.
- Preserve the two frontend API surfaces:
  - Ollama-compatible: `/api/generate`, `/api/chat`, `/api/tags`, `/api/show`
  - OpenAI-compatible: `/v1/chat/completions`, `/v1/models`
- Preserve the web/API utility routes: `/`, `/logs`, `/logs/details`, `/logs/download`, `/api/logs`, `/api/logs/{id}`, and `/health`.
- OpenAI backend support lives in `backend/openai.go`; OpenAI-compatible frontend support lives in `handlers/openai_frontend.go`.
- Text injection and tool blacklist behavior apply to both chat frontend endpoints: `/api/chat` and `/v1/chat/completions`.
- Request sanitization and stream override behavior must remain consistent across `/api/generate`, `/api/chat`, and `/v1/chat/completions`.
- `ollama_overrides.thinking` affects chat requests only and is materialized only by the Ollama backend.
- `gemma_4_fix.enabled` affects OpenAI-backend chat processing only; keep the mitigation isolated and update both design documents in `docs/` if its behavior changes.
- Config parsing intentionally rejects unknown TOML keys.
- `config.toml` and `data/` are local runtime files and are ignored by git. Keep changes in `config.toml.example` when documenting default config shape.

## Testing Guidance

- Run `CGO_ENABLED=0 go test ./...` before finishing code changes.
- Add focused tests when changing request translation, handler response formats, or database behavior.
- Existing useful test areas:
  - `backend/convert_test.go` for tool-call conversion to OpenAI format.
  - `database/sqlite_test.go` for SQLite round trips.
  - `handlers/openai_frontend_test.go` for OpenAI-compatible frontend behavior.
  - `handlers/llmlog_test.go` for log formatting.
  - `handlers/request_sanitization_test.go` and `handlers/stream_override_test.go` for the spy-backend, cross-endpoint parity test pattern used for request-mutation features.
  - `handlers/chat_features_test.go` and `handlers/ollama_thinking_override_test.go` for shared chat mutations and Ollama thinking behavior.
  - `backend/gemma4_fix_test.go` for the optional Gemma 4 stream mitigation.
  - `handlers/logs_api_test.go` and `handlers/web_test.go` for JSON and HTML log routes.

## Commit Guidance

- When an AI agent creates a commit, add a `Co-authored-by` trailer naming the model currently running. Do not use a fixed model name inherited from an earlier session. Use the form `Co-authored-by: <current model name> <noreply@openai.com>` (for example, `Co-authored-by: GPT-5 <noreply@openai.com>`).

## Documentation Guidance

- Keep `README.md`, `DOCKER.md`, and `AGENTS.md` aligned with `config.toml.example`, `go.mod`, Docker files, scripts, and exposed routes.
- If endpoints are added or changed, update both the API endpoint list and the project structure section in `README.md`.
- If release targets change, update the README release section and check `.goreleaser.yml` / `.github/workflows/release.yml`.
