package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mxcoppell/mdp/web"
)

// TOCEntry represents a heading in the table of contents.
type TOCEntry struct {
	ID       string     `json:"id"`
	Text     string     `json:"text"`
	Level    int        `json:"level"`
	Children []TOCEntry `json:"children,omitempty"`
}

// Config holds server configuration.
type Config struct {
	Port                int
	Theme               string
	HTML                string
	TOC                 []TOCEntry
	Filename            string
	FilePath            string
	BaseDir             string // Directory to serve local files from (empty = no local serving)
	ShowTOC             bool
	HasMath             bool
	HasMermaid          bool
	WordCount           int
	NoWatch             bool
	Verbose             bool
	DisableAutoShutdown bool // In host mode, the host manages window lifecycle
}

// Server is the HTTP server for mdp.
type Server struct {
	cfg        Config
	mu         sync.RWMutex
	html       string
	toc        []TOCEntry
	hasMath    bool
	hasMermaid bool
	wordCount  int
	srv        *http.Server
	ws         *WSHub
	cancel     context.CancelFunc
	done       chan struct{}
	listener   net.Listener
	OnShutdown func()
}

// New creates a new Server.
func New(cfg Config) *Server {
	s := &Server{
		cfg:        cfg,
		html:       cfg.HTML,
		toc:        cfg.TOC,
		hasMath:    cfg.HasMath,
		hasMermaid: cfg.HasMermaid,
		wordCount:  cfg.WordCount,
		done:       make(chan struct{}),
	}
	s.ws = newWSHub(s)
	return s
}

// Start starts the HTTP server and returns the listening address.
func (s *Server) Start(ctx context.Context) (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	addr := "127.0.0.1:0"
	if s.cfg.Port > 0 {
		addr = fmt.Sprintf("127.0.0.1:%d", s.cfg.Port)
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		cancel()
		return "", fmt.Errorf("listen: %w", err)
	}
	s.listener = ln

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.srv = &http.Server{Handler: mux}

	go func() {
		defer close(s.done)
		if err := s.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Printf("mdp: server error: %v\n", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutCancel()
		_ = s.srv.Shutdown(shutCtx)
	}()

	return ln.Addr().String(), nil
}

// Wait blocks until the server has shut down.
func (s *Server) Wait() {
	<-s.done
}

// Shutdown initiates graceful shutdown.
func (s *Server) Shutdown() {
	if s.cancel != nil {
		s.cancel()
	}
}

// UpdateContent updates the rendered HTML and broadcasts to WebSocket clients.
func (s *Server) UpdateContent(html string, toc []TOCEntry, hasMath, hasMermaid bool, wordCount int) {
	s.mu.Lock()
	s.html = html
	s.toc = toc
	s.hasMath = hasMath
	s.hasMermaid = hasMermaid
	s.wordCount = wordCount
	s.mu.Unlock()

	// Broadcast update message as JSON
	msg := map[string]any{
		"type":       "update",
		"html":       html,
		"toc":        toc,
		"hasMath":    hasMath,
		"hasMermaid": hasMermaid,
		"wordCount":  wordCount,
	}
	data, _ := json.Marshal(msg)
	s.ws.broadcast(string(data))
}

// Addr returns the listener address, or empty string if not started.
func (s *Server) Addr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Serve embedded static files
	staticFS, _ := fs.Sub(web.Assets, "static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// HTML template
	mux.HandleFunc("/", s.handleIndex)

	// API endpoints
	mux.HandleFunc("/api/shutdown", s.handleShutdown)

	// WebSocket
	mux.HandleFunc("/ws", s.ws.handleWS)
}

func (s *Server) serveLocalFile(w http.ResponseWriter, r *http.Request) {
	if s.cfg.BaseDir == "" {
		http.NotFound(w, r)
		return
	}

	cleaned := path.Clean(r.URL.Path)
	resolved := filepath.Join(s.cfg.BaseDir, filepath.FromSlash(cleaned))

	// Prevent directory traversal outside BaseDir
	if !strings.HasPrefix(resolved, s.cfg.BaseDir+string(filepath.Separator)) && resolved != s.cfg.BaseDir {
		http.NotFound(w, r)
		return
	}

	info, err := os.Stat(resolved)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, resolved)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		s.serveLocalFile(w, r)
		return
	}

	tmplData, err := web.Assets.ReadFile("templates/index.html")
	if err != nil {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("index").Parse(string(tmplData))
	if err != nil {
		http.Error(w, "template parse error", http.StatusInternalServerError)
		return
	}

	s.mu.RLock()
	html := s.html
	toc := s.toc
	hasMath := s.hasMath
	hasMermaid := s.hasMermaid
	wordCount := s.wordCount
	s.mu.RUnlock()

	tocJSON, _ := json.Marshal(toc)

	data := map[string]any{
		"Theme":      s.cfg.Theme,
		"HTML":       template.HTML(html),
		"TOC":        template.JS(tocJSON),
		"Filename":   s.cfg.Filename,
		"FilePath":   s.cfg.FilePath,
		"ShowTOC":    s.cfg.ShowTOC,
		"HasMath":    hasMath,
		"HasMermaid": hasMermaid,
		"WordCount":  wordCount,
		"NoWatch":    s.cfg.NoWatch,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, data)
}

func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("shutting down"))

	go func() {
		time.Sleep(100 * time.Millisecond)
		if s.OnShutdown != nil {
			s.OnShutdown()
		}
		s.Shutdown()
	}()
}
