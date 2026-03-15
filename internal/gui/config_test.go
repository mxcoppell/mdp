package gui

import (
	"testing"
	"time"
)

func TestConfig_WriteAndRead(t *testing.T) {
	original := Config{
		Port:       8080,
		Theme:      "dark",
		HTML:       "<h1>Hello</h1>",
		Filename:   "test.md",
		FilePath:   "./test.md",
		NoWatch:    true,
		ShowTOC:    true,
		HasMath:    true,
		HasMermaid: false,
		WordCount:  42,
	}

	path, err := WriteConfig(original)
	if err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	read, err := ReadConfig(path)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	if read.Port != original.Port {
		t.Errorf("Port: got %d, want %d", read.Port, original.Port)
	}
	if read.Theme != original.Theme {
		t.Errorf("Theme: got %q, want %q", read.Theme, original.Theme)
	}
	if read.HTML != original.HTML {
		t.Errorf("HTML: got %q, want %q", read.HTML, original.HTML)
	}
	if read.Filename != original.Filename {
		t.Errorf("Filename: got %q, want %q", read.Filename, original.Filename)
	}
	if read.NoWatch != original.NoWatch {
		t.Errorf("NoWatch: got %v, want %v", read.NoWatch, original.NoWatch)
	}
	if read.ShowTOC != original.ShowTOC {
		t.Errorf("ShowTOC: got %v, want %v", read.ShowTOC, original.ShowTOC)
	}
	if read.HasMath != original.HasMath {
		t.Errorf("HasMath: got %v, want %v", read.HasMath, original.HasMath)
	}
	if read.HasMermaid != original.HasMermaid {
		t.Errorf("HasMermaid: got %v, want %v", read.HasMermaid, original.HasMermaid)
	}
	if read.WordCount != original.WordCount {
		t.Errorf("WordCount: got %d, want %d", read.WordCount, original.WordCount)
	}
}

func TestConfig_AllFields(t *testing.T) {
	original := Config{
		Port:       9090,
		Theme:      "system",
		HTML:       "<p>test</p>",
		TOC:        []TOCEntry{{ID: "intro", Text: "Intro", Level: 1}},
		RawContent: "# Intro\nContent",
		Filename:   "readme.md",
		FilePath:   "/path/to/readme.md",
		WatchFiles: []string{"/path/to/readme.md"},
		Poll:       500 * time.Millisecond,
		NoWatch:    false,
		ShowTOC:    true,
		Browser:    true,
		HasMath:    true,
		HasMermaid: true,
		WordCount:  100,
	}

	path, err := WriteConfig(original)
	if err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	read, err := ReadConfig(path)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	if len(read.TOC) != 1 {
		t.Errorf("TOC length: got %d, want 1", len(read.TOC))
	}
	if read.TOC[0].ID != "intro" {
		t.Errorf("TOC[0].ID: got %q, want %q", read.TOC[0].ID, "intro")
	}
	if len(read.WatchFiles) != 1 {
		t.Errorf("WatchFiles length: got %d, want 1", len(read.WatchFiles))
	}
	if read.Browser != original.Browser {
		t.Errorf("Browser: got %v, want %v", read.Browser, original.Browser)
	}
	if read.Poll != original.Poll {
		t.Errorf("Poll: got %v, want %v", read.Poll, original.Poll)
	}
}
