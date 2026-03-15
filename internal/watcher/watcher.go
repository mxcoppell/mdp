package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches for file changes and sends updated content.
type Watcher interface {
	Start(ctx context.Context) error
	Content() <-chan string
}

// FileWatcher uses fsnotify to watch a file's directory for changes.
type FileWatcher struct {
	path    string
	content chan string
	w       *fsnotify.Watcher
}

// NewFileWatcher creates a watcher using fsnotify.
// It watches the parent directory to handle atomic saves (vim, VS Code).
func NewFileWatcher(path string) (*FileWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("fsnotify: %w", err)
	}

	dir := filepath.Dir(path)
	if err := w.Add(dir); err != nil {
		w.Close()
		return nil, fmt.Errorf("watch %s: %w", dir, err)
	}

	return &FileWatcher{
		path:    path,
		content: make(chan string, 1),
		w:       w,
	}, nil
}

func (fw *FileWatcher) Start(ctx context.Context) error {
	defer fw.w.Close()
	defer close(fw.content)

	target := filepath.Base(fw.path)
	var debounce *time.Timer

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-fw.w.Events:
			if !ok {
				return nil
			}
			if filepath.Base(event.Name) != target {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			if debounce != nil {
				debounce.Stop()
			}
			debounce = time.AfterFunc(100*time.Millisecond, func() {
				data, err := os.ReadFile(fw.path)
				if err != nil {
					return
				}
				select {
				case fw.content <- string(data):
				default:
				}
			})
		case err, ok := <-fw.w.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(os.Stderr, "md-preview-cli: watcher error: %v\n", err)
		}
	}
}

func (fw *FileWatcher) Content() <-chan string {
	return fw.content
}

// PollWatcher uses stat-based polling for environments where fsnotify doesn't work.
type PollWatcher struct {
	path     string
	interval time.Duration
	content  chan string
}

// NewPollWatcher creates a stat-based polling watcher.
func NewPollWatcher(path string, interval time.Duration) *PollWatcher {
	return &PollWatcher{
		path:     path,
		interval: interval,
		content:  make(chan string, 1),
	}
}

func (pw *PollWatcher) Start(ctx context.Context) error {
	defer close(pw.content)

	var lastMod time.Time
	info, err := os.Stat(pw.path)
	if err == nil {
		lastMod = info.ModTime()
	}

	ticker := time.NewTicker(pw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			info, err := os.Stat(pw.path)
			if err != nil {
				continue
			}
			if info.ModTime().After(lastMod) {
				lastMod = info.ModTime()
				data, err := os.ReadFile(pw.path)
				if err != nil {
					continue
				}
				select {
				case pw.content <- string(data):
				default:
				}
			}
		}
	}
}

func (pw *PollWatcher) Content() <-chan string {
	return pw.content
}

// NoopWatcher is used for stdin input where there's nothing to watch.
type NoopWatcher struct {
	content chan string
}

// NewNoopWatcher creates a watcher that never sends updates.
func NewNoopWatcher() *NoopWatcher {
	return &NoopWatcher{content: make(chan string)}
}

func (nw *NoopWatcher) Start(ctx context.Context) error {
	<-ctx.Done()
	close(nw.content)
	return nil
}

func (nw *NoopWatcher) Content() <-chan string {
	return nw.content
}
