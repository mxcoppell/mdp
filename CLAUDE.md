# md-preview-cli

A CLI tool that renders markdown files in a native frameless window. Designed for terminal-only agents like Claude Code.

## Architecture

- **Language**: Go with CGO (webview + Cocoa frameless)
- **Two-process model**: CLI spawns GUI subprocess via `--internal-gui=<config.json>`, exits immediately
- **Rendering**: Server-side Goldmark (CommonMark + GFM) + Chroma syntax highlighting
- **Client-side**: KaTeX (math) and Mermaid.js (diagrams) lazy-loaded only when present
- **Single binary**: All HTML/CSS/JS/fonts embedded via `go:embed`

## Key Directories

- `cmd/` — CLI flag parsing, stdin detection, GUI spawning
- `internal/gui/` — GUI process lifecycle, webview, frameless macOS window (CGO)
- `internal/server/` — HTTP server, WebSocket hub, auto-shutdown
- `internal/renderer/` — Goldmark markdown→HTML pipeline, TOC extraction, math/mermaid placeholders
- `internal/watcher/` — fsnotify + stat-based polling file watchers
- `web/` — Embedded assets (HTML template, JS, CSS, vendored libraries)
- `testdata/` — Test markdown files

## Build

```bash
# Download vendored JS dependencies (first time only)
./scripts/download-deps.sh

# Build
go build -o md-preview-cli .

# Run
./md-preview-cli testdata/full-spec.md
echo "# Hello" | ./md-preview-cli
```

## Testing

```bash
go test ./...
go test -v ./internal/renderer/...
```

## Conventions

- Follow the mermaid-preview-cli patterns for GUI, server, watcher, WebSocket
- Post-process HTML rather than custom Goldmark AST transformers where simpler
- Theme system uses CSS custom properties with system/light/dark modes
- All keyboard shortcuts are vim-inspired (j/k, n/p, g g/G)
