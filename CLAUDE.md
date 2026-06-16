# CLAUDE.md

Guidance for working in this repo.

## Overview

`moexoptcalc` is a Go **library** (module `github.com/acidsailor/moexoptcalc`,
Go 1.26) — a client for the public **MOEX ISS Options Calculator** API
(`https://iss.moex.com/iss/apps/option-calc/v1`). It returns the official
greeks, IV, theoretical prices, and portfolio/margin metrics MOEX publishes
for FORTS options. The API is public (no auth).

Library only — no `cmd/`, no `main`, no binary.

## Common commands

The repo uses [Task](https://taskfile.dev) (`taskfile.yml`). Codegen tasks
need Docker; build/test/lint need the Go toolchain.

```sh
task test     # go test ./...  (NOTE: includes live API calls to iss.moex.com)
task vet      # go vet ./...
task lint     # golangci-lint fmt + golangci-lint run --fix  (mutates files)
task ci       # read-only: golangci-lint fmt --diff + golangci-lint run
task check    # composite: lint (mutating) + vet + test
task tidy     # go mod tidy
```

There is no build target — it's a library; use `go build ./...` if needed.
Run a single test with `go test -run TestName ./...`. The test suite hits the
live MOEX API; `go test -short ./...` skips the live calls.

Codegen pipeline (Docker required, no local codegen toolchain):

```sh
task spec     # download spec/spec-upstream.json from iss.moex.com (committed)
task overlay  # Speakeasy: apply spec/overlay.yml -> spec/spec.json (oapi-codegen input)
task deref    # Redocly bundle --dereferenced (from upstream) -> spec/spec-deref.json (MCP input)
task gen      # overlay + oapi-codegen v2.5.0 -> models.gen.go (fails on ANY warning)
task update   # copier update from the go-scaffolds template
```

CI runs the reusable workflow `acidsailor/go-scaffolds/.github/workflows/go-ci.yml@v1`.

## Architecture

Single public package `moexoptcalc` at the repo root (no `internal/`). The
generated models and the hand-written transport share that one package.

- **Models are code-generated** by oapi-codegen v2 into `models.gen.go` as
  plain Go structs (config `oapi-codegen.yaml`, `generate: models: true`).
  Do not edit `models.gen.go` by hand. Optional/nullable fields are pointers
  (nil = absent); required fields are values. Read methods return the struct
  directly.
- **Transport** wraps the models. `client.go` defines `Client`, holding a
  `*restkit.Client` (`github.com/acidsailor/restkit`). `New(endpoint,
  ...ClientOption)` is the entry point; returns `*ConfigError` on an empty
  endpoint. The 13 per-operation methods are thin wrappers over generic
  `restkit.Do[T]`. The option type is `ClientOption` (not `Option`, the
  generated model for an option instrument); the only option is
  `WithHTTPClient`.
- **`params.go`** holds the per-method `*Params` structs. Path params are
  required strings; optional query filters are pointers; date filters are
  `*Date`. The `Date` (date-only, `2006-01-02`) and `DateTime` (parsed through
  `Europe/Moscow`, wire form `2006-01-02T15:04:05`) types are package-local
  `time.Time` wrappers with custom JSON marshaling. `tzdata.go` embeds
  `time/tzdata` so the Moscow zone resolves on any host (~450 KB binary cost).
- **`errors.go`** re-exports restkit's three typed errors as package aliases:
  `*ConfigError` (construction), `*RequestError` (per-call failure, with an
  `Op` stage field), `*ResponseError` (non-2xx, with `StatusCode` and raw
  `Body`). No sentinel vars — match with `errors.As`; wrapped causes survive
  `errors.Is` (e.g. `context.Canceled`).
- **OTel**: `restkit.New` wraps the transport with `otelhttp`, emitting
  standard `http.client.*` spans/metrics via the global OTel providers. This
  package adds no instruments; instrumentation is opt-in by setting the global
  providers before `New`.
- **`spec/`**: `spec.go` (`package spec`) embeds `spec-deref.json` as
  `SpecDerefJSON` for downstream consumers (e.g. an MCP wrapper). The pipeline
  branches from `spec-upstream.json` into two committed artifacts: `spec.json`
  (overlaid, ref-based — oapi-codegen input, not embedded) and `spec-deref.json`
  (dereferenced OpenAPI 3.1 — embedded).

## Conventions & gotchas

- **`task gen` fails on any oapi-codegen warning** (greps output for `warn`).
  The overlay relabels the spec to `3.0.3` for the codegen branch to avoid the
  3.1 "not yet supported" warning; the deref/MCP branch stays 3.1.
- **No `oapi-codegen/runtime` dependency** in `models.gen.go`, by design.
  Keep it that way: `oapi-codegen.yaml` excludes `HTTPValidationError` /
  `ValidationError` (the only union schemas), and date fields map to the local
  `Date`/`DateTime` types via `x-go-type` rather than `openapi_types.Date`.
- **Case-sensitive asset codes.** The wire form of `asset_code` is not
  canonical (e.g. `Si` vs `SI`); round-trip it through `ListAssets`/`GetAsset`
  rather than constructing it locally.
- **Moscow-zoned timestamps.** `format: date-time` fields arrive without a TZ
  but are Moscow-local; the `DateTime` type handles this — don't treat the
  wall-clock as UTC.
- Linting via golangci-lint v2 (`.golangci.yaml`): standard set + `modernize`;
  formatters gofumpt (extra rules) and golines (80-col).
