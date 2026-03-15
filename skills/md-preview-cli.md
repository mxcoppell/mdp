---
name: md-preview-cli
description: Preview, render, display, or show markdown files in a native frameless window — supports stdin fire-and-forget, live file preview with reload, TOC sidebar, search, math, and diagrams
---

# md-preview-cli

Preview markdown files in a native frameless macOS window. Fire-and-forget from the terminal.

## Usage

```bash
# File preview with live reload
md-preview-cli README.md

# Pipe from stdin
cat doc.md | md-preview-cli

# Multiple files
md-preview-cli a.md b.md

# With options
md-preview-cli --theme dark --toc README.md
md-preview-cli --browser README.md  # open in system browser
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `j`/`k` | Scroll down/up |
| `Space`/`Shift+Space` | Page down/up |
| `g g` / `G` | Top / Bottom |
| `n`/`p` | Next/prev heading |
| `N`/`P` | Next/prev section (H1/H2) |
| `]` | Toggle TOC sidebar |
| `/` or `Cmd+F` | Search |
| `T` | Cycle theme |
| `+`/`-`/`0` | Zoom |
| `?` | Help overlay |
| `Esc` | Close search or window |

## Features

- CommonMark + GFM (tables, task lists, strikethrough, autolinks)
- Footnotes, definition lists, emoji, typographer
- Syntax-highlighted code blocks (200+ languages via Chroma)
- KaTeX math rendering (lazy-loaded)
- Mermaid diagram rendering (lazy-loaded)
- GitHub-style admonitions (NOTE, TIP, WARNING, CAUTION, IMPORTANT)
- Table of contents sidebar with scroll spy
- In-content search with match navigation
- Live reload on file changes
- Dark/light/system theme with persistence
