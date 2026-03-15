package renderer

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Result holds the output of rendering markdown.
type Result struct {
	HTML       string
	TOC        []TOCEntry
	HasMath    bool
	HasMermaid bool
	WordCount  int
}

// TOCEntry represents a heading in the table of contents.
type TOCEntry struct {
	ID       string     `json:"id"`
	Text     string     `json:"text"`
	Level    int        `json:"level"`
	Children []TOCEntry `json:"children,omitempty"`
}

// Render converts markdown content to HTML with all extensions enabled.
func Render(content string) Result {
	// Strip YAML front matter
	content = stripFrontMatter(content)

	// Detect math and mermaid before rendering
	hasMath := detectMath(content)
	hasMermaid := detectMermaid(content)

	// Protect math delimiters from Goldmark parsing
	var mathBlocks []string
	var mathInlines []string
	content, mathBlocks, mathInlines = protectMath(content)

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			extension.DefinitionList,
			extension.Typographer,
			emoji.Emoji,
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(true),
				),
				highlighting.WithWrapperRenderer(
					func(w util.BufWriter, ctx highlighting.CodeBlockContext, entering bool) {
						if entering {
							lang, _ := ctx.Language()
							if len(lang) > 0 {
								_, _ = io.WriteString(w, `<pre><code class="language-`+string(lang)+` chroma">`)
							} else {
								_, _ = io.WriteString(w, `<pre><code class="chroma">`)
							}
						} else {
							_, _ = io.WriteString(w, `</code></pre>`)
						}
					},
				),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)

	source := []byte(content)
	doc := md.Parser().Parse(text.NewReader(source))

	// Extract TOC from AST before rendering
	toc := extractTOC(doc, source)

	// Render to HTML
	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, source, doc); err != nil {
		return Result{HTML: "<p>Error rendering markdown</p>"}
	}

	output := buf.String()

	// Post-process: restore math placeholders
	output = restoreMath(output, mathBlocks, mathInlines)

	// Post-process: convert mermaid code blocks to placeholder divs
	output = convertMermaidBlocks(output)

	// Post-process: convert admonitions (GitHub-style > [!NOTE] etc.)
	output = convertAdmonitions(output)

	// Add language labels to code blocks
	output = addCodeBlockLabels(output)

	return Result{
		HTML:       output,
		TOC:        toc,
		HasMath:    hasMath,
		HasMermaid: hasMermaid,
		WordCount:  countWords(content),
	}
}

// stripFrontMatter removes YAML front matter delimited by ---
func stripFrontMatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}
	end := strings.Index(content[3:], "\n---")
	if end == -1 {
		return content
	}
	return strings.TrimLeft(content[end+7:], "\n")
}

var (
	mathInlineRe = regexp.MustCompile(`(?:^|[^$\\])\$([^\$\n]+?)\$(?:[^$]|$)`)
	mathBlockRe  = regexp.MustCompile(`\$\$[\s\S]+?\$\$`)
)

func detectMath(content string) bool {
	return mathBlockRe.MatchString(content) || mathInlineRe.MatchString(content)
}

func detectMermaid(content string) bool {
	return strings.Contains(content, "```mermaid")
}

// protectMath replaces math delimiters with numbered placeholders before Goldmark parsing.
func protectMath(content string) (string, []string, []string) {
	var blocks []string
	var inlines []string

	// Protect block math first ($$...$$)
	blockCounter := 0
	content = mathBlockRe.ReplaceAllStringFunc(content, func(match string) string {
		inner := match[2 : len(match)-2]
		blocks = append(blocks, inner)
		placeholder := fmt.Sprintf("\n\nMATHBLOCK%d\n\n", blockCounter)
		blockCounter++
		return placeholder
	})

	// Protect inline math ($...$)
	inlineCounter := 0
	content = regexp.MustCompile(`\$([^\$\n]+?)\$`).ReplaceAllStringFunc(content, func(match string) string {
		inner := match[1 : len(match)-1]
		// Don't match things like $5 or price$tag
		if len(inner) == 0 {
			return match
		}
		inlines = append(inlines, inner)
		placeholder := fmt.Sprintf("MATHINLINE%d", inlineCounter)
		inlineCounter++
		return placeholder
	})

	return content, blocks, inlines
}

// restoreMath replaces math placeholders with proper HTML elements.
func restoreMath(html string, blocks, inlines []string) string {
	for i, inner := range blocks {
		placeholder := fmt.Sprintf("MATHBLOCK%d", i)
		// The placeholder may be wrapped in <p> tags
		html = strings.ReplaceAll(html, "<p>"+placeholder+"</p>",
			`<div class="math-block">`+escapeHTML(inner)+`</div>`)
		html = strings.ReplaceAll(html, placeholder,
			`<div class="math-block">`+escapeHTML(inner)+`</div>`)
	}

	for i, inner := range inlines {
		placeholder := fmt.Sprintf("MATHINLINE%d", i)
		html = strings.ReplaceAll(html, placeholder,
			`<span class="math-inline">`+escapeHTML(inner)+`</span>`)
	}

	return html
}

// convertMermaidBlocks replaces <pre><code class="language-mermaid">...</code></pre>
// with <div class="mermaid-placeholder">...</div>
var mermaidBlockRe = regexp.MustCompile(`<pre><code class="language-mermaid[^"]*">([\s\S]*?)</code></pre>`)

func convertMermaidBlocks(html string) string {
	return mermaidBlockRe.ReplaceAllStringFunc(html, func(match string) string {
		sub := mermaidBlockRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		source := strings.TrimSpace(sub[1])
		return `<div class="mermaid-placeholder">` + source + `</div>`
	})
}

// convertAdmonitions converts GitHub-style admonitions in blockquotes.
// > [!NOTE] → <div class="admonition note">
var admonitionRe = regexp.MustCompile(`<blockquote>\s*<p>\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]`)

func convertAdmonitions(html string) string {
	// Find all blockquotes that start with [!TYPE]
	return admonitionRe.ReplaceAllStringFunc(html, func(match string) string {
		sub := admonitionRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		adType := strings.ToLower(sub[1])
		title := strings.ToUpper(adType[:1]) + adType[1:]

		icons := map[string]string{
			"note":      "&#8505;&#65039;",  // ℹ️
			"tip":       "&#128161;",         // 💡
			"important": "&#10071;",          // ❗
			"warning":   "&#9888;&#65039;",   // ⚠️
			"caution":   "&#128308;",         // 🔴
		}
		icon := icons[adType]

		return `<div class="admonition ` + adType + `"><p class="admonition-title"><span class="admonition-icon">` + icon + `</span> ` + title
	})
}

// addCodeBlockLabels adds language labels to highlighted code blocks.
var codeBlockFullRe = regexp.MustCompile(`<pre[^>]*><code class="language-(\w+)[^"]*">[\s\S]*?</code></pre>`)

func addCodeBlockLabels(html string) string {
	return codeBlockFullRe.ReplaceAllStringFunc(html, func(match string) string {
		sub := codeBlockFullRe.FindStringSubmatch(match)
		if len(sub) < 2 || sub[1] == "mermaid" {
			return match
		}
		lang := sub[1]
		return `<div class="code-block-wrapper"><span class="code-lang-label">` + lang + `</span>` + match + `</div>`
	})
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func countWords(content string) int {
	count := 0
	inWord := false
	for _, r := range content {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				count++
				inWord = true
			}
		} else {
			inWord = false
		}
	}
	return count
}
