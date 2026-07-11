# Code Review

Reviewed: 2026-07-11

## Verdict

This project is not garbage. It is a small, understandable Go service with sensible package boundaries, a notably good collection of compatibility tests, pure-Go SQLite, strict TOML parsing, context-aware upstream requests, and careful separation of client-facing and backend-facing streaming behavior.

It is best described as a solid personal/LAN tool that is not yet hardened as an Internet-facing service. The largest risks are around its trust boundary and failure handling, not its core request translation. In particular, the default listener is network-wide while every request and response is persisted and made available through unauthenticated log routes.

## Scope and checks

The review covered server/configuration wiring, both backend adapters, all frontend handlers, logging and SQLite access, middleware, the web UI, deployment files, and representative tests.

Checks run:

- `GOCACHE=/tmp/llm_proxy_go_build CGO_ENABLED=0 go test ./...` — passed
- `GOCACHE=/tmp/llm_proxy_go_build CGO_ENABLED=0 go vet ./...` — passed
- `GOCACHE=/tmp/llm_proxy_go_build go test -race ./...` — could not complete in the restricted review environment because `httptest` was not permitted to open a loopback listener; packages reached before that failure reported no race

This was a source review, not a penetration test or a live interoperability test against real OpenAI/vLLM/Ollama servers.

## What is good

- The repository is easy to navigate. Config, backends, handlers, models, database code, and middleware have clear ownership.
- The streaming invariant is documented and tested: an upstream stream override does not silently change the response shape requested by the client.
- Cross-endpoint parity tests for request mutations are exactly the right testing pattern for this proxy.
- Unknown TOML keys are rejected, enum-like settings are validated, and configuration behavior has focused tests.
- SQL values are parameterized; ordering is allow-listed rather than interpolated from arbitrary input.
- Templates use `html/template`, which gives the log viewer an appropriate escaping baseline.
- Backend HTTP calls use request contexts and a configured client timeout.
- SQLite uses a pure-Go driver, WAL, and per-connection busy timeouts. Database tests cover real round trips.
- Embedded UI assets and the dependency-free chat client keep distribution simple.
- The unusual Gemma 4 workaround is isolated and unusually well documented instead of leaking special cases throughout the main flow.

## Findings

### P0 — Decide and enforce the service trust boundary

The default host is `0.0.0.0`, and there is no authentication or authorization on inference, configuration, web log, log download, or JSON log endpoints. Independently of the stdout logging flags, handlers persist frontend and backend request/response bodies to SQLite. `/api/logs/{id}`, `/logs/details`, and `/logs/download` expose those bodies. Prompts commonly contain private text, credentials pasted by users, proprietary code, and tool results.

That combination is unsafe on an untrusted network. CORS being disabled does not protect non-browser clients and is not authentication.

Recommended fix: default to loopback, document that the service must not be exposed directly, and add optional authentication that covers every route except perhaps `/health`. If remote use is supported, require a deliberate configuration choice and recommend TLS through a trusted reverse proxy. Add configurable body retention/redaction; a stdout logging switch should not be confused with database privacy.

Relevant code: `config/config.go`, `main.go`, `handlers/chat.go`, `handlers/generate.go`, `handlers/openai_frontend.go`, `handlers/logs_api.go`, and `handlers/web.go`.

### P1 — Bound inbound requests and in-memory response capture

The three completion handlers call `io.ReadAll(r.Body)` without a limit. Responses are also retained several times while being proxied: aggregate text, a slice of every converted chunk, captured frontend output, and raw upstream output. Large prompts, base64 images, or long generations can therefore consume disproportionate memory and disk, and concurrent requests multiply the cost.

Recommended fix: add a configurable request-body limit using `http.MaxBytesReader`, return `413`, and test all three endpoints. Stream database logging to a bounded representation or cap/truncate captured bodies with explicit metadata such as `truncated: true`. Avoid retaining both every structured chunk and a reconstructed copy unless raw-body logging is enabled.

Relevant code: `handlers/chat.go`, `handlers/generate.go`, `handlers/openai_frontend.go`, `backend/openai.go`, and `backend/ollama.go`.

### P1 — Do not turn upstream decode/stream failures into successful completions

Several backend goroutines cannot return an error after the initial HTTP request succeeds. Non-streaming handlers silently return no channel item when reading, JSON decoding, or response-shape validation fails. Streaming handlers skip malformed events and, on scanner error or unexpected EOF, often synthesize a normal `Done: true` response. The frontend can consequently return HTTP 200 with an empty or partial answer and log it as successful.

The channel-only backend contract has no way to represent a terminal error. Logging scanner errors with `fmt.Printf` does not inform the caller.

Recommended fix: change the stream item/result contract to carry terminal errors (or provide a separate terminal error result), distinguish a clean protocol terminator from EOF, validate that non-streaming responses contain a choice, and propagate failures to handler logging. Before response headers are committed, return an appropriate 502; after streaming starts, terminate the stream in the closest protocol-compatible way and record the failure.

Relevant code: `backend/openai.go` (`handleStreamingCompletion`, `handleNonStreamingCompletion`, `handleStreamingChat`, `handleNonStreamingChat`) and the scanner loops in `backend/ollama.go`.

### P1 — Add HTTP server timeouts and use graceful shutdown

The frontend `http.Server` sets only `Addr` and `Handler`. There are no header/read/idle timeouts or maximum header size. A slow client can hold connections and goroutines cheaply. Shutdown calls `server.Close()`, which immediately closes active connections, including generations in flight, rather than allowing a bounded drain.

Recommended fix: configure `ReadHeaderTimeout`, `IdleTimeout`, and `MaxHeaderBytes`. Treat `WriteTimeout` carefully because legitimate LLM streams may be long. Use `server.Shutdown` with a deadline, then fall back to `Close`, and stop accepting work before closing shared resources.

Relevant code: `main.go`.

### P1 — Validate numeric configuration before using it

Port, backend timeout, cleanup interval, and maximum request count are not consistently range-checked. A negative cleanup interval reaches `time.NewTicker` and can panic at startup. Invalid ports fail late. A negative backend timeout has surprising behavior. The struct comments say zero means cleanup disabled/unlimited, but `Load` replaces zero with defaults, so those documented states cannot actually be configured.

Recommended fix: explicitly validate ranges and resolve the zero-value ambiguity. If zero must be distinguishable from omission, use pointer fields or decode into an intermediate config with presence tracking. Add tests for negative values, out-of-range ports, and explicit zero semantics.

Relevant code: `config/config.go` and `main.go`.

### P2 — Return proxy-appropriate errors without leaking backend bodies

Backend failures are generally returned as `500 Internal Server Error`, and some error strings include the complete upstream response body. This both misclassifies upstream failures (normally 502/504) and can disclose backend internals to clients. Model-list fallback behavior can also hide a real backend outage behind a fabricated `default` model.

Recommended fix: introduce typed backend errors with safe public messages, internal causes, upstream status, and timeout classification. Map them to 502/504 as appropriate, log diagnostic bodies only in protected/bounded logs, and reconsider whether `/v1/models` failure should be synthetic or explicit.

Relevant code: `backend/openai.go`, `backend/ollama.go`, `handlers/chat.go`, `handlers/generate.go`, `handlers/openai_frontend.go`, and `handlers/models.go`.

### P2 — Make observability safer and more operationally useful

Logs use the standard unstructured logger, scanner failures use `fmt.Printf`, request IDs are absent, and health always returns OK without checking whether SQLite is usable. Database writes happen synchronously at the end of requests, adding log latency and coupling successful inference to storage contention (although write failure does not fail the response).

Recommended fix: use structured logging with a request/correlation ID, consistent error paths, and secret-aware redaction. Split liveness from readiness and have readiness ping SQLite; optionally test backend reachability with a separate, cached signal. Measure backend latency separately from total handler and database-log latency.

### P2 — Add CI and a few high-value failure tests

The local suite is good, but there is no visible CI workflow in this checkout. The most important missing tests exercise hostile or broken I/O rather than happy-path conversion: oversized bodies, truncated SSE/NDJSON, invalid non-streaming JSON, client cancellation/backpressure, handler behavior when the backend terminates without `Done`, authentication, and graceful shutdown.

Recommended fix: run formatting checks, `go vet`, `CGO_ENABLED=0 go test ./...`, and `go test -race ./...` in CI. Add the failure-injection tests above and an end-to-end test using `httptest.Server` across frontend handler and backend adapter.

### P3 — Small maintainability improvements

- The three completion handlers duplicate body reading, logging, aggregation, and error mapping. Extract only the stable plumbing; keep protocol formatting explicit.
- `homeData` is a `map[string]interface{}` and `WebHandler.config` is `interface{}`. A typed view model would make configuration/UI drift compile-visible.
- `generateToolCallID` ignores the error from `rand.Read`. It is extraordinarily unlikely to fail, but the function should return an error or use a clearly documented fallback.
- Add `X-Content-Type-Options: nosniff` and a restrictive CSP to the web UI. These are defense-in-depth alongside authentication, not substitutes for it.
- Pin a non-root user in the runtime container and document expected ownership of the mounted database directory.

## Prioritized todo list

- [ ] **P0:** Define local-only versus remote deployment support; default to loopback and protect all sensitive routes with optional authentication.
- [ ] **P0:** Document that the database stores full prompt/response bodies regardless of stdout logging flags; add retention/redaction controls.
- [ ] **P1:** Add configurable request size limits and bounded/truncated response capture across all three completion endpoints.
- [ ] **P1:** Redesign backend stream results so read, decode, truncation, and protocol errors reach handlers and logs.
- [ ] **P1:** Add frontend HTTP timeouts/header limits and bounded graceful shutdown.
- [ ] **P1:** Validate numeric config and make explicit zero values behave as documented.
- [ ] **P2:** Introduce safe typed backend errors and map upstream failures to 502/504 without returning raw bodies.
- [ ] **P2:** Add request IDs, structured/redacted logs, and meaningful readiness checks.
- [ ] **P2:** Add CI with tests, vet, formatting, and the race detector.
- [ ] **P2:** Add tests for oversized input, malformed/truncated upstream data, cancellation, missing terminal chunks, and shutdown.
- [ ] **P3:** Use typed web view data, reduce stable handler duplication, handle random-ID errors, add security headers, and run the container as non-root.

## Suggested delivery order

First establish whether this is intentionally local-only. If it is, changing the default bind address plus prominent documentation sharply reduces immediate risk. If remote access is a requirement, authentication and TLS deployment guidance come first. Next implement bounded input/log capture and honest upstream error propagation; those changes prevent the most damaging reliability failures. Server timeouts, config validation, and graceful shutdown are comparatively contained follow-ups. Finish with observability, CI failure cases, and cleanup refactors.
