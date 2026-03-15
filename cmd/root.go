package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"golang.org/x/term"

	"github.com/mxcoppell/md-preview-cli/internal/gui"
	"github.com/mxcoppell/md-preview-cli/internal/renderer"
	"github.com/mxcoppell/md-preview-cli/internal/version"
)

const maxStdinSize = 10 * 1024 * 1024 // 10MB

// Config holds all CLI configuration.
type Config struct {
	Port    int
	Theme   string
	NoWatch bool
	Poll    time.Duration
	ShowTOC bool
	Browser bool
	Wait    bool
	Files   []string
	IsStdin bool
}

func Execute() int {
	cfg, err := parseFlags(os.Args[1:], os.Stdin)
	if err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		fmt.Fprintf(os.Stderr, "md-preview-cli: error: %v\n", err)
		return 1
	}

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "md-preview-cli: error: %v\n", err)
		return 2
	}
	return 0
}

func parseFlags(args []string, stdin *os.File) (Config, error) {
	var cfg Config
	var showVersion bool

	fs := flag.NewFlagSet("md-preview-cli", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.IntVar(&cfg.Port, "port", 0, "")
	fs.IntVar(&cfg.Port, "p", 0, "")
	fs.StringVar(&cfg.Theme, "theme", "system", "")
	fs.StringVar(&cfg.Theme, "t", "system", "")
	fs.BoolVar(&cfg.NoWatch, "no-watch", false, "")
	fs.BoolVar(&cfg.NoWatch, "w", false, "")
	fs.DurationVar(&cfg.Poll, "poll", 0, "")
	fs.BoolVar(&cfg.ShowTOC, "toc", false, "")
	fs.BoolVar(&cfg.Browser, "browser", false, "")
	fs.BoolVar(&cfg.Wait, "wait", false, "")
	fs.BoolVar(&showVersion, "version", false, "")
	fs.BoolVar(&showVersion, "v", false, "")

	var showHelp bool
	fs.BoolVar(&showHelp, "help", false, "")
	fs.BoolVar(&showHelp, "h", false, "")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if showHelp {
		printHelp(os.Stdout)
		return Config{}, flag.ErrHelp
	}

	if showVersion {
		fmt.Fprintf(os.Stdout, "md-preview-cli %s\n", version.Version)
		return Config{}, flag.ErrHelp
	}

	// Validate theme
	switch cfg.Theme {
	case "dark", "light", "system":
	default:
		return Config{}, fmt.Errorf("invalid theme %q: must be dark, light, or system", cfg.Theme)
	}

	// Check for file arguments first, then fall back to stdin detection
	remaining := fs.Args()
	if len(remaining) >= 1 {
		cfg.Files = remaining
	} else if !term.IsTerminal(int(stdin.Fd())) {
		cfg.IsStdin = true
		cfg.NoWatch = true
		cfg.Files = []string{""}
	} else {
		printHelp(os.Stderr)
		return Config{}, fmt.Errorf("no input file specified")
	}

	return cfg, nil
}

func run(cfg Config) error {
	for _, file := range cfg.Files {
		var content string
		var filename string

		if cfg.IsStdin {
			data, err := io.ReadAll(io.LimitReader(os.Stdin, maxStdinSize))
			if err != nil {
				return fmt.Errorf("reading stdin: %w", err)
			}
			content = string(data)
			filename = "stdin"
		} else {
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("reading %s: %w", file, err)
			}
			content = string(data)
			filename = filepath.Base(file)
		}

		// Render markdown to HTML
		result := renderer.Render(content)

		// Build watch file list
		var watchFiles []string
		if !cfg.NoWatch && !cfg.IsStdin {
			absPath, err := filepath.Abs(file)
			if err != nil {
				return fmt.Errorf("resolving path: %w", err)
			}
			watchFiles = []string{absPath}
		}

		guiCfg := gui.Config{
			Port:       cfg.Port,
			Theme:      cfg.Theme,
			HTML:       result.HTML,
			TOC:        convertTOC(result.TOC),
			RawContent: content,
			Filename:   filename,
			FilePath:   file,
			WatchFiles: watchFiles,
			Poll:       cfg.Poll,
			NoWatch:    cfg.NoWatch,
			ShowTOC:    cfg.ShowTOC,
			Browser:    cfg.Browser,
			HasMath:    result.HasMath,
			HasMermaid: result.HasMermaid,
			WordCount:  result.WordCount,
		}

		if err := spawnGUI(guiCfg, cfg.Wait); err != nil {
			return err
		}
	}

	return nil
}

func spawnGUI(cfg gui.Config, wait bool) error {
	tmpPath, err := gui.WriteConfig(cfg)
	if err != nil {
		return fmt.Errorf("writing GUI config: %w", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	cmd := exec.Command(exePath, "--internal-gui="+tmpPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("spawning GUI: %w", err)
	}

	if wait {
		return cmd.Wait()
	}

	return nil
}

func convertTOC(entries []renderer.TOCEntry) []gui.TOCEntry {
	result := make([]gui.TOCEntry, len(entries))
	for i, e := range entries {
		result[i] = gui.TOCEntry{
			ID:       e.ID,
			Text:     e.Text,
			Level:    e.Level,
			Children: convertTOC(e.Children),
		}
	}
	return result
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, `USAGE:
    md-preview-cli [FLAGS] [FILE.md ...]

ARGUMENTS:
    FILE    One or more markdown files (.md, .markdown)
            Each file opens in its own preview window

FLAGS:
    -p, --port PORT       Server port (default: auto-select available)
    -t, --theme THEME     dark | light | system (default: system)
    -w, --no-watch        Disable file watching
        --poll INTERVAL   Stat-based polling fallback (e.g. 500ms)
        --toc             Show table of contents sidebar on startup
        --browser         Force open in system browser (no native window)
        --wait            Block until preview window is closed
    -v, --version         Print version
    -h, --help            Print help

STDIN:
    cat README.md | md-preview-cli
    echo "# Hello World" | md-preview-cli

AGENT TOOL USAGE:
    Pipe markdown to stdin. The CLI opens a native preview window
    and exits immediately (exit code 0).

    Example from an LLM agent:
        cat README.md | md-preview-cli

    For file-based preview with live reload:
        md-preview-cli README.md

KEYBOARD SHORTCUTS (in preview window):
    j/k         Scroll down/up        n/p  Next/prev heading
    ]           Toggle TOC sidebar
    Cmd/Ctrl+F  Search                T    Toggle theme
    +/-         Zoom in/out           0    Reset zoom
    h           Show shortcuts         Esc  Close

EXIT CODES: 0 = success, 1 = argument error, 2 = runtime error
`)
}
