package gui

import (
	"context"
	"fmt"
	"os"
	osexec "os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/mxcoppell/mdp/internal/renderer"
	"github.com/mxcoppell/mdp/internal/server"
	"github.com/mxcoppell/mdp/internal/watcher"
)

// Run is the GUI process entry point. It reads the config from the temp file,
// starts an HTTP server, optionally starts file watchers, creates a frameless
// webview window, and runs the event loop until the window is closed.
func Run(cfgPath string) error {
	cfg, err := ReadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := server.New(server.Config{
		Port:       cfg.Port,
		Theme:      cfg.Theme,
		HTML:       cfg.HTML,
		TOC:        toServerTOC(cfg.TOC),
		Filename:   cfg.Filename,
		FilePath:   cfg.FilePath,
		BaseDir:    dirFromFilePath(cfg.FilePath),
		ShowTOC:    cfg.ShowTOC,
		HasMath:    cfg.HasMath,
		HasMermaid: cfg.HasMermaid,
		WordCount:  cfg.WordCount,
		NoWatch:    cfg.NoWatch,
		Verbose:    cfg.Verbose,
	})

	addr, err := srv.Start(ctx)
	if err != nil {
		return fmt.Errorf("starting server: %w", err)
	}
	url := fmt.Sprintf("http://%s", addr)
	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "mdp: listening on %s (%s)\n", url, cfg.Filename)
	}

	// Browser mode: open in system browser instead of webview
	if cfg.Browser {
		if err := openBrowser(url); err != nil {
			return fmt.Errorf("opening browser: %w", err)
		}
		startFileWatchers(ctx, cfg, srv)
		srv.Wait()
		return nil
	}

	// Start file watchers
	startFileWatchers(ctx, cfg, srv)

	// Create webview window
	w := createWindow(url)
	defer w.Destroy()

	// Wire server shutdown → webview terminate
	srv.OnShutdown = func() {
		w.Terminate()
	}

	// Handle SIGINT and SIGTERM gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		w.Terminate()
	}()

	// If the server exits for any reason (e.g. auto-shutdown after 30s
	// with no WebSocket clients), terminate the webview so the process
	// doesn't linger invisibly (no dock icon).
	go func() {
		srv.Wait()
		w.Terminate()
	}()

	// Schedule frameless styling before starting the event loop.
	scheduleFrameless(w.Window())

	// Run webview event loop (blocks until window is closed)
	w.Run()

	// Clean up
	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "mdp: shutting down\n")
	}
	srv.Shutdown()
	srv.Wait()
	return nil
}

func startFileWatchers(ctx context.Context, cfg Config, srv *server.Server) {
	if cfg.NoWatch || len(cfg.WatchFiles) == 0 {
		return
	}

	for _, file := range cfg.WatchFiles {
		absPath, err := filepath.Abs(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mdp: resolve path error (%s): %v\n", file, err)
			continue
		}

		var w watcher.Watcher
		if cfg.Poll > 0 {
			w = watcher.NewPollWatcher(absPath, cfg.Poll)
		} else {
			w, err = watcher.NewFileWatcher(absPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "mdp: watcher error (%s): %v\n", file, err)
				continue
			}
		}

		go func() {
			if err := w.Start(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "mdp: watcher error: %v\n", err)
			}
		}()

		go func() {
			for newContent := range w.Content() {
				result := renderer.Render(newContent)
				srv.UpdateContent(result.HTML, rendererTOCToServer(result.TOC), result.HasMath, result.HasMermaid, result.WordCount)
			}
		}()
	}
}

func toServerTOC(entries []TOCEntry) []server.TOCEntry {
	result := make([]server.TOCEntry, len(entries))
	for i, e := range entries {
		result[i] = server.TOCEntry{
			ID:       e.ID,
			Text:     e.Text,
			Level:    e.Level,
			Children: toServerTOC(e.Children),
		}
	}
	return result
}

func dirFromFilePath(p string) string {
	if p == "" {
		return ""
	}
	return filepath.Dir(p)
}

func openBrowser(url string) error {
	return osexec.Command("open", url).Start()
}
