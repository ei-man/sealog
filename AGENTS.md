# AGENTS.md

## Project Overview

Sealog is a local log viewer for structured application logs. It can read logs from stdin or query Azure Application Insights traces, then display the results in a Svelte UI with field selection, a detail panel, backend filtering, and shared time-range controls.

The backend owns log parsing, query parsing, filtering, time range handling, and Azure KQL translation. The frontend sends query text unchanged and renders state received over WebSocket.

## Commands

```bash
make dev          # Run Vite frontend and Go backend together
make dev-example  # Run the .NET example and pipe it through Sealog
make build        # Build frontend and production ./sealog binary
make check        # Run frontend type checks and Go tests
make clean        # Remove build artifacts
```

Useful focused checks:

```bash
go test ./...
cd frontend && npm run check
cd frontend && npm run build
go build -o sealog .
```

When testing with a live Sealog process, stop the process before finishing the task.

Homebrew setup:

- `Formula/sealog.rb` is a source-build Homebrew formula.
- This repository doubles as the tap; use `brew tap ei-man/sealog https://github.com/ei-man/sealog`.
- `.github/workflows/update-homebrew-formula.yml` updates `Formula/sealog.rb` with a tagged source URL and SHA-256 when tags matching `v*` are pushed.
- The formula keeps `head` support for `brew install --HEAD sealog`.

## Runtime Modes

- Stdin mode: `some-command | sealog`
- Azure Application Insights mode: `./sealog --app-insights`
- Azure mode uses Azure CLI credentials and expects the user to have run `az login`.
- Azure v1 queries Application Insights `traces` only.

## Architecture

Go backend:

- `main.go`: CLI flags, embedded frontend serving, stdin reader, HTTP setup.
- `app.go`: application state, source selection, query application, time range refresh, Azure orchestration.
- `hub.go`: `/ws` protocol, reconnect snapshots, broadcast fan-out.
- `parser.go`: JSON/raw line parsing, nested field storage, dot-path flattening, event timestamp detection.
- `query.go`: Sealog query lexer/parser/evaluator and KQL translation.
- `time_range.go`: preset/custom time range validation and containment.
- `azure.go`: Azure CLI integration, App Insights resource listing, traces query response parsing.

Frontend:

- Svelte 5 app under `frontend/src/`.
- `frontend/src/lib/websocket.ts` is the only app protocol client.
- `frontend/src/lib/stores/logs.svelte.ts` holds server-driven log state.
- Main components are `FilterBar`, `TimeRangeControl`, `FieldSelector`, `LogTable`, and `DetailPanel`.

Generated files:

- `frontend/dist/`, `frontend/node_modules/`, `sealog`, and .NET `bin`/`obj` output are generated. Do not hand-edit them.

## Query Language

Supported forms:

- `field=value`
- `field="value"`
- `field~="regex"`
- `AND`, `OR`, `NOT`
- parentheses
- dot-path fields such as `person.name`

Boolean operators must be explicit. Adjacent conditions like `level=error message~="timeout"` are syntax errors. Exact comparisons are case-insensitive. Regex filters are compiled case-insensitively.

## Time and Fields

- Default time range is `1h`.
- Presets are `15m`, `30m`, `1h`, `3h`, `6h`, `12h`, `24h`, and `3d`; custom ranges require both start and end.
- Stdin logs get an `ingested_at` metadata field.
- Stdin time filtering uses a parseable event timestamp first, then falls back to `ingested_at`.
- Nested JSON is preserved for details and copied entries; flattened dot paths are used for columns, field selection, and queries.
- App Insights `customDimensions` are expanded into fields, and JSON-looking dimension values are parsed into nested fields when possible.

## WebSocket Protocol

`/ws` is the main app protocol. Client commands include:

- `query.apply`
- `timeRange.set`
- `source.select`
- `azure.resources.list`
- `azure.resource.select`

Server messages include:

- `source`
- `timeRange`
- `fields`
- `results.reset`
- `results.stats`
- `line`
- `query.error`
- `azure.resources`
- `loading`

There is no `/api/fields`; fields are delivered over WebSocket.

## Development Notes

- Keep backend query behavior canonical. Do not reintroduce client-side query parsing/filtering.
- Keep stdin and Azure behavior aligned through the same query and time range models.
- Use existing Svelte component/store patterns before adding new abstractions.
- Run `make check` after Go or frontend behavior changes. Documentation-only edits do not need tests.
- The project is licensed under AGPLv3; keep license references aligned with `LICENSE`.
