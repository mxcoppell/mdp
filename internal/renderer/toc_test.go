package renderer

import (
	"testing"
)

func TestTOC_BasicHeadings(t *testing.T) {
	result := Render("# H1\n## H2\n### H3")
	if len(result.TOC) != 3 {
		t.Fatalf("expected 3 TOC entries, got %d", len(result.TOC))
	}
	if result.TOC[0].Level != 1 || result.TOC[0].Text != "H1" {
		t.Errorf("expected H1 entry, got: %+v", result.TOC[0])
	}
	if result.TOC[1].Level != 2 || result.TOC[1].Text != "H2" {
		t.Errorf("expected H2 entry, got: %+v", result.TOC[1])
	}
	if result.TOC[2].Level != 3 || result.TOC[2].Text != "H3" {
		t.Errorf("expected H3 entry, got: %+v", result.TOC[2])
	}
}

func TestTOC_NoHeadings(t *testing.T) {
	result := Render("Just a paragraph.\n\nAnother paragraph.")
	if len(result.TOC) != 0 {
		t.Errorf("expected empty TOC, got %d entries", len(result.TOC))
	}
}

func TestTOC_HeadingWithCode(t *testing.T) {
	result := Render("## `func()` Usage")
	if len(result.TOC) != 1 {
		t.Fatalf("expected 1 TOC entry, got %d", len(result.TOC))
	}
	if result.TOC[0].Text != "func() Usage" {
		t.Errorf("expected 'func() Usage', got: %q", result.TOC[0].Text)
	}
}

func TestTOC_DuplicateHeadings(t *testing.T) {
	result := Render("## Setup\n\ntext\n\n## Setup")
	if len(result.TOC) != 2 {
		t.Fatalf("expected 2 TOC entries, got %d", len(result.TOC))
	}
	// IDs should be unique
	if result.TOC[0].ID == result.TOC[1].ID {
		t.Errorf("expected unique IDs, both are: %s", result.TOC[0].ID)
	}
}

func TestTOC_HasIDs(t *testing.T) {
	result := Render("# Introduction\n## Getting Started")
	for _, entry := range result.TOC {
		if entry.ID == "" {
			t.Errorf("expected non-empty ID for entry: %+v", entry)
		}
	}
}

func TestTOC_SkippedLevels(t *testing.T) {
	result := Render("# H1\n### H3")
	if len(result.TOC) != 2 {
		t.Fatalf("expected 2 TOC entries, got %d", len(result.TOC))
	}
	if result.TOC[0].Level != 1 {
		t.Errorf("expected level 1, got %d", result.TOC[0].Level)
	}
	if result.TOC[1].Level != 3 {
		t.Errorf("expected level 3, got %d", result.TOC[1].Level)
	}
}
