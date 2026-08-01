# Code Review

Reviewed: 2026-07-31
Candidate: v0.2.23

## Verdict

The fork is reasonable to release for its documented trust boundary. Its API
surface is focused, the compatibility behavior is well tested, and the service
has no unnecessary adapter namespaces or speculative endpoints.

This is an Ollama/OpenAI protocol bridge with operational request inspection,
not a general model gateway. It should run on a trusted network or behind an
authenticated TLS reverse proxy.

## Surface Review

The retained surface has a concrete consumer or operator use:

- Ollama compatibility: `/api/chat`, `/api/generate`, `/api/tags`, `/api/show`
- Basic OpenAI compatibility: `/v1/chat/completions`, `/v1/models`
- Operations: `/health`, the web log viewer, and the JSON logs API

`/api/embed` remains intentionally absent because there is no tested consumer
contract for it. Client authentication, model routing, and backend management
also remain outside this proxy's scope.

## v0.2.23 Hardening

- Completion request bodies are bounded across both frontend protocols and
  oversized requests fail with HTTP 413 before backend work begins.
- Malformed JSON is no longer printed with its raw body unless raw request
  logging is explicitly enabled.
- Invalid ports, negative timeouts, and invalid database cleanup values fail at
  configuration load. Explicit zero cleanup values now behave as documented.
- The HTTP server limits header-read and idle time and drains active requests
  for up to 30 seconds during shutdown.
- Obsolete legacy completion handlers were removed.
- Release images run as UID/GID 10001, use the documented port in their health
  check, and carry the repository's Apache-2.0 license metadata.
- CI and release gates now include formatting, vet, shuffled tests, and the race
  detector before publication.

## Verification

- Full Go suite passed with shuffled ordering.
- Full Go suite passed ten repeated runs.
- Race detector passed three repeated runs.
- `go vet` and Staticcheck passed.
- `govulncheck` found no reachable vulnerabilities.
- GoReleaser configuration, archive contents, checksums, and embedded build
  identity passed.
- The non-root image passed health, model discovery, streamed chat,
  non-streamed schema generation, compatibility-hint validation, metadata-only
  logging, and oversized-request smokes.

## Bounded Follow-up

These are known limits, not hidden release claims:

- The service has no built-in authentication or TLS. All inference and log
  routes share the deployment trust boundary.
- Successful responses can still be retained in multiple in-memory forms for
  logging. Large response capture should be bounded in a later focused change.
- The backend channel contract cannot always distinguish a clean stream ending
  from a malformed upstream termination after response streaming has begun.
- `/health` is liveness only; it does not assert backend readiness.
- Automated release artifacts currently target Linux x86-64.
