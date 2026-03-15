#!/usr/bin/env bash
set -euo pipefail

# Downloads vendored JS/CSS at pinned versions for embedding into the binary.
# Run this BEFORE `go build`. The files are committed to the repo.

VENDOR_DIR="web/static/vendor"
mkdir -p "${VENDOR_DIR}" "${VENDOR_DIR}/fonts"

# ─── Mermaid.js ────────────────────────────────────────────
MERMAID_VERSION="11.4.1"
MERMAID_URL="https://cdn.jsdelivr.net/npm/mermaid@${MERMAID_VERSION}/dist/mermaid.min.js"

echo "Downloading mermaid.js v${MERMAID_VERSION}..."
curl -sL "${MERMAID_URL}" -o "${VENDOR_DIR}/mermaid.min.js"

if grep -q "mermaid" "${VENDOR_DIR}/mermaid.min.js"; then
    echo "  OK: mermaid.min.js ($(wc -c < "${VENDOR_DIR}/mermaid.min.js" | tr -d ' ') bytes)"
else
    echo "  ERROR: downloaded file does not appear to be mermaid.js"
    rm -f "${VENDOR_DIR}/mermaid.min.js"
    exit 1
fi

# ─── KaTeX ─────────────────────────────────────────────────
KATEX_VERSION="0.16.11"
KATEX_BASE="https://cdn.jsdelivr.net/npm/katex@${KATEX_VERSION}/dist"

echo "Downloading KaTeX v${KATEX_VERSION}..."
curl -sL "${KATEX_BASE}/katex.min.js" -o "${VENDOR_DIR}/katex.min.js"
curl -sL "${KATEX_BASE}/katex.min.css" -o "${VENDOR_DIR}/katex.min.css"

# Download KaTeX fonts
KATEX_FONTS=(
    "KaTeX_AMS-Regular.woff2"
    "KaTeX_Caligraphic-Bold.woff2"
    "KaTeX_Caligraphic-Regular.woff2"
    "KaTeX_Fraktur-Bold.woff2"
    "KaTeX_Fraktur-Regular.woff2"
    "KaTeX_Main-Bold.woff2"
    "KaTeX_Main-BoldItalic.woff2"
    "KaTeX_Main-Italic.woff2"
    "KaTeX_Main-Regular.woff2"
    "KaTeX_Math-BoldItalic.woff2"
    "KaTeX_Math-Italic.woff2"
    "KaTeX_SansSerif-Bold.woff2"
    "KaTeX_SansSerif-Italic.woff2"
    "KaTeX_SansSerif-Regular.woff2"
    "KaTeX_Script-Regular.woff2"
    "KaTeX_Size1-Regular.woff2"
    "KaTeX_Size2-Regular.woff2"
    "KaTeX_Size3-Regular.woff2"
    "KaTeX_Size4-Regular.woff2"
    "KaTeX_Typewriter-Regular.woff2"
)

for font in "${KATEX_FONTS[@]}"; do
    curl -sL "${KATEX_BASE}/fonts/${font}" -o "${VENDOR_DIR}/fonts/${font}"
done

# Fix font paths in katex.min.css (fonts/ → /static/vendor/fonts/)
sed -i '' 's|url(fonts/|url(/static/vendor/fonts/|g' "${VENDOR_DIR}/katex.min.css"

if grep -q "katex" "${VENDOR_DIR}/katex.min.js"; then
    echo "  OK: katex.min.js ($(wc -c < "${VENDOR_DIR}/katex.min.js" | tr -d ' ') bytes)"
    echo "  OK: katex.min.css ($(wc -c < "${VENDOR_DIR}/katex.min.css" | tr -d ' ') bytes)"
    echo "  OK: ${#KATEX_FONTS[@]} font files"
else
    echo "  ERROR: downloaded file does not appear to be katex.js"
    exit 1
fi

# ─── github-markdown-css ───────────────────────────────────
GHMD_VERSION="5.8.1"
GHMD_URL="https://cdn.jsdelivr.net/npm/github-markdown-css@${GHMD_VERSION}/github-markdown.css"

echo "Downloading github-markdown-css v${GHMD_VERSION}..."
curl -sL "${GHMD_URL}" -o "${VENDOR_DIR}/github-markdown.css"

if grep -q "markdown-body" "${VENDOR_DIR}/github-markdown.css"; then
    echo "  OK: github-markdown.css ($(wc -c < "${VENDOR_DIR}/github-markdown.css" | tr -d ' ') bytes)"
else
    echo "  ERROR: downloaded file does not appear to be github-markdown-css"
    exit 1
fi

echo ""
echo "All dependencies downloaded to ${VENDOR_DIR}/"
echo "Run 'go build' to embed them into the binary."
