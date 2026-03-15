package gui

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// TOCEntry represents a heading in the table of contents.
type TOCEntry struct {
	ID       string     `json:"id"`
	Text     string     `json:"text"`
	Level    int        `json:"level"`
	Children []TOCEntry `json:"children,omitempty"`
}

// Config holds configuration passed from CLI mode to GUI mode via a temp JSON file.
type Config struct {
	Port       int           `json:"port"`
	Theme      string        `json:"theme"`
	HTML       string        `json:"html"`
	TOC        []TOCEntry    `json:"toc"`
	RawContent string        `json:"raw_content"`
	Filename   string        `json:"filename"`
	FilePath   string        `json:"file_path"`
	WatchFiles []string      `json:"watch_files,omitempty"`
	Poll       time.Duration `json:"poll,omitempty"`
	NoWatch    bool          `json:"no_watch"`
	ShowTOC    bool          `json:"show_toc"`
	Browser    bool          `json:"browser"`
	HasMath    bool          `json:"has_math"`
	HasMermaid bool          `json:"has_mermaid"`
	WordCount  int           `json:"word_count"`
}

// WriteConfig serializes cfg to a temp JSON file and returns the path.
func WriteConfig(cfg Config) (string, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}

	f, err := os.CreateTemp("", "md-preview-cli-gui-*.json")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("write config: %w", err)
	}

	return f.Name(), nil
}

// ReadConfig reads the config from the given path and deletes the file.
func ReadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	os.Remove(path)

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}
	return cfg, nil
}
