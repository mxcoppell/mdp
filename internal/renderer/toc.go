package renderer

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
)

// extractTOC walks the Goldmark AST and builds a flat list of TOCEntry items.
// The frontend builds the nested tree from the flat list using heading levels.
func extractTOC(doc ast.Node, source []byte) []TOCEntry {
	var entries []TOCEntry

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		heading, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		// Extract heading text
		var text bytes.Buffer
		for child := heading.FirstChild(); child != nil; child = child.NextSibling() {
			extractText(child, source, &text)
		}

		// Get auto-generated ID
		id, ok := heading.AttributeString("id")
		if !ok {
			return ast.WalkContinue, nil
		}

		entries = append(entries, TOCEntry{
			ID:    string(id.([]byte)),
			Text:  text.String(),
			Level: heading.Level,
		})

		return ast.WalkContinue, nil
	})

	return entries
}

// extractText recursively extracts plain text from AST nodes.
func extractText(n ast.Node, source []byte, buf *bytes.Buffer) {
	if n.Kind() == ast.KindText {
		t := n.(*ast.Text)
		buf.Write(t.Segment.Value(source))
		return
	}
	if n.Kind() == ast.KindCodeSpan {
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Kind() == ast.KindText {
				t := child.(*ast.Text)
				buf.Write(t.Segment.Value(source))
			}
		}
		return
	}
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		extractText(child, source, buf)
	}
}
