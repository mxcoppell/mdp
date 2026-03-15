package renderer

import (
	"strings"
	"testing"
)

func TestRender_BasicMarkdown(t *testing.T) {
	result := Render("# Hello")
	if !strings.Contains(result.HTML, "<h1") || !strings.Contains(result.HTML, "Hello") {
		t.Errorf("expected <h1>Hello</h1>, got: %s", result.HTML)
	}
}

func TestRender_GFMTable(t *testing.T) {
	md := "| A | B |\n|---|---|\n| 1 | 2 |"
	result := Render(md)
	if !strings.Contains(result.HTML, "<table>") {
		t.Errorf("expected <table>, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "<th>") {
		t.Errorf("expected <th>, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "<td>") {
		t.Errorf("expected <td>, got: %s", result.HTML)
	}
}

func TestRender_GFMStrikethrough(t *testing.T) {
	result := Render("~~deleted~~")
	if !strings.Contains(result.HTML, "<del>") {
		t.Errorf("expected <del>, got: %s", result.HTML)
	}
}

func TestRender_GFMTaskList(t *testing.T) {
	result := Render("- [x] Done\n- [ ] Todo")
	if !strings.Contains(result.HTML, "checkbox") {
		t.Errorf("expected checkbox, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "checked") {
		t.Errorf("expected checked attribute, got: %s", result.HTML)
	}
}

func TestRender_GFMAutolink(t *testing.T) {
	result := Render("Visit https://example.com for info.")
	if !strings.Contains(result.HTML, `<a href="https://example.com"`) {
		t.Errorf("expected autolink, got: %s", result.HTML)
	}
}

func TestRender_Footnotes(t *testing.T) {
	md := "Text[^1]\n\n[^1]: Footnote content"
	result := Render(md)
	if !strings.Contains(result.HTML, "footnote") {
		t.Errorf("expected footnote HTML, got: %s", result.HTML)
	}
}

func TestRender_DefinitionList(t *testing.T) {
	md := "Term\n: Definition"
	result := Render(md)
	if !strings.Contains(result.HTML, "<dl>") {
		t.Errorf("expected <dl>, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "<dt>") {
		t.Errorf("expected <dt>, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "<dd>") {
		t.Errorf("expected <dd>, got: %s", result.HTML)
	}
}

func TestRender_HeadingIDs(t *testing.T) {
	result := Render("## Setup Guide")
	if !strings.Contains(result.HTML, `id="setup-guide"`) {
		t.Errorf("expected heading ID, got: %s", result.HTML)
	}
}

func TestRender_CodeBlock_Highlighted(t *testing.T) {
	md := "```go\nfunc main() {}\n```"
	result := Render(md)
	if !strings.Contains(result.HTML, "language-go") {
		t.Errorf("expected language-go class, got: %s", result.HTML)
	}
}

func TestRender_CodeBlock_NoLang(t *testing.T) {
	md := "```\nplain code\n```"
	result := Render(md)
	if !strings.Contains(result.HTML, "<pre>") {
		t.Errorf("expected <pre>, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "plain code") {
		t.Errorf("expected plain code content, got: %s", result.HTML)
	}
}

func TestRender_MathInline(t *testing.T) {
	result := Render("The equation $E=mc^2$ is famous.")
	if !strings.Contains(result.HTML, `class="math-inline"`) {
		t.Errorf("expected math-inline class, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "E=mc^2") {
		t.Errorf("expected math content preserved, got: %s", result.HTML)
	}
}

func TestRender_MathBlock(t *testing.T) {
	result := Render("$$\n\\sum_{i=0}^n i\n$$")
	if !strings.Contains(result.HTML, `class="math-block"`) {
		t.Errorf("expected math-block class, got: %s", result.HTML)
	}
}

func TestRender_MathDetection(t *testing.T) {
	result := Render("$x + y$")
	if !result.HasMath {
		t.Error("expected HasMath=true")
	}

	result = Render("No math here")
	if result.HasMath {
		t.Error("expected HasMath=false")
	}
}

func TestRender_MermaidBlock(t *testing.T) {
	md := "```mermaid\ngraph TD\n    A-->B\n```"
	result := Render(md)
	if !strings.Contains(result.HTML, `class="mermaid-placeholder"`) {
		t.Errorf("expected mermaid-placeholder, got: %s", result.HTML)
	}
}

func TestRender_MermaidDetection(t *testing.T) {
	md := "```mermaid\ngraph TD\n    A-->B\n```"
	result := Render(md)
	if !result.HasMermaid {
		t.Error("expected HasMermaid=true")
	}

	result = Render("No mermaid here")
	if result.HasMermaid {
		t.Error("expected HasMermaid=false")
	}
}

func TestRender_Admonition_Note(t *testing.T) {
	md := "> [!NOTE]\n> This is a note."
	result := Render(md)
	if !strings.Contains(result.HTML, `class="admonition note"`) {
		t.Errorf("expected admonition note, got: %s", result.HTML)
	}
}

func TestRender_Admonition_Warning(t *testing.T) {
	md := "> [!WARNING]\n> Be careful."
	result := Render(md)
	if !strings.Contains(result.HTML, `class="admonition warning"`) {
		t.Errorf("expected admonition warning, got: %s", result.HTML)
	}
}

func TestRender_Admonition_AllTypes(t *testing.T) {
	types := []string{"NOTE", "TIP", "IMPORTANT", "WARNING", "CAUTION"}
	for _, typ := range types {
		md := "> [!" + typ + "]\n> Content"
		result := Render(md)
		expected := `class="admonition ` + strings.ToLower(typ) + `"`
		if !strings.Contains(result.HTML, expected) {
			t.Errorf("expected %s admonition, got: %s", typ, result.HTML)
		}
	}
}

func TestRender_FrontMatter(t *testing.T) {
	md := "---\ntitle: Test\nauthor: Bob\n---\n\n# Content"
	result := Render(md)
	if strings.Contains(result.HTML, "title:") || strings.Contains(result.HTML, "author:") {
		t.Errorf("front matter should be stripped, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "Content") {
		t.Errorf("expected content after front matter, got: %s", result.HTML)
	}
}

func TestRender_EmptyInput(t *testing.T) {
	result := Render("")
	if result.HTML != "" {
		t.Errorf("expected empty output for empty input, got: %q", result.HTML)
	}
}

func TestRender_LargeDocument(t *testing.T) {
	// Generate a large document
	var sb strings.Builder
	sb.WriteString("# Large Document\n\n")
	for i := 0; i < 500; i++ {
		sb.WriteString("## Section ")
		sb.WriteString(strings.Repeat("x", 5))
		sb.WriteString("\n\nParagraph of text. ")
		sb.WriteString("More text here. ")
		sb.WriteString("And some **bold** and *italic* content.\n\n")
	}

	result := Render(sb.String())
	if len(result.HTML) == 0 {
		t.Error("expected non-empty output for large document")
	}
	if len(result.TOC) == 0 {
		t.Error("expected non-empty TOC for large document")
	}
}

func TestRender_WordCount(t *testing.T) {
	result := Render("Hello world. This is a test.")
	if result.WordCount < 5 {
		t.Errorf("expected word count >= 5, got %d", result.WordCount)
	}
}

func TestRender_CodeBlockLabel(t *testing.T) {
	md := "```python\nprint('hello')\n```"
	result := Render(md)
	if !strings.Contains(result.HTML, `class="code-lang-label"`) {
		t.Errorf("expected code-lang-label, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "python") {
		t.Errorf("expected python label, got: %s", result.HTML)
	}
}
