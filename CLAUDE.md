# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Sealog is a real-time log viewer that reads JSON logs from stdin and displays them in a web UI. Pipe any command's output through `sealog` to view structured logs with filtering, column selection, and live updates.

Usage: `some-command | sealog`

## Commands

```bash
make dev      # Run frontend (Vite) + backend (Go) together for development
make build    # Build production binary with embedded frontend
make check    # Run svelte-check and go test
make clean    # Remove build artifacts
```

## Architecture

**Go backend** (`main.go`, `hub.go`, `parser.go`):
- Reads stdin line-by-line, parses JSON (or treats as raw text)
- Stores lines in a ring buffer (`RingBuffer` in main.go, default 50k lines)
- WebSocket hub broadcasts new lines to all connected clients
- Embeds `frontend/dist/*` into the binary via `//go:embed`

**Svelte 5 frontend** (`frontend/src/`):
- Connects via WebSocket to `/ws`, receives `line` and `fields` messages
- `logs` store (`lib/stores/logs.svelte`) holds all state
- Virtual scrolling in `LogTable.svelte` for large datasets

**Data flow**: stdin → `ParseLine()` → `RingBuffer.Add()` → `Hub.BroadcastLine()` → WebSocket → Svelte store → UI

## Development Notes

- Frontend dev server runs on `127.0.0.1:5173`, proxied to Go backend
- The `--no-open` flag prevents auto-opening browser (used in `make dev`)
- Non-JSON lines are stored with `raw: true` and displayed as-is
