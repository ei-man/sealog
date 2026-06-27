package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"sync"
)

//go:embed frontend/dist/*
var embeddedFiles embed.FS

type RingBuffer struct {
	mu     sync.RWMutex
	lines  []LogLine
	start  int
	count  int
	nextID int64
	fields map[string]struct{}
}

func NewRingBuffer(maxLines int) *RingBuffer {
	if maxLines < 1 {
		maxLines = 1
	}
	return &RingBuffer{
		lines:  make([]LogLine, maxLines),
		fields: make(map[string]struct{}),
	}
}

func (b *RingBuffer) Add(fields map[string]any, raw bool) (LogLine, bool, []string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	line := LogLine{
		ID:     b.nextID,
		Fields: cloneFields(fields),
		Raw:    raw,
	}

	index := (b.start + b.count) % len(b.lines)
	if b.count == len(b.lines) {
		index = b.start
		b.start = (b.start + 1) % len(b.lines)
	} else {
		b.count++
	}
	b.lines[index] = line

	changed := false
	for field := range fields {
		if _, ok := b.fields[field]; !ok {
			b.fields[field] = struct{}{}
			changed = true
		}
	}

	return line, changed, sortedFields(b.fields)
}

func (b *RingBuffer) Lines() []LogLine {
	b.mu.RLock()
	defer b.mu.RUnlock()

	lines := make([]LogLine, 0, b.count)
	for i := 0; i < b.count; i++ {
		index := (b.start + i) % len(b.lines)
		line := b.lines[index]
		line.Fields = cloneFields(line.Fields)
		lines = append(lines, line)
	}
	return lines
}

func (b *RingBuffer) Fields() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return sortedFields(b.fields)
}

func cloneFields(fields map[string]any) map[string]any {
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		out[key] = value
	}
	return out
}

func sortedFields(fields map[string]struct{}) []string {
	names := make([]string, 0, len(fields))
	for field := range fields {
		names = append(names, field)
	}
	sort.Slice(names, func(i, j int) bool {
		if names[i] == names[j] {
			return false
		}
		return names[i] < names[j]
	})
	return names
}

func main() {
	port := flag.Int("port", 7100, "HTTP server port")
	noOpen := flag.Bool("no-open", false, "suppress auto-opening browser")
	maxLines := flag.Int("max-lines", 50000, "max lines to keep in memory")
	flag.Parse()

	buffer := NewRingBuffer(*maxLines)
	hub := NewHub()

	mux, err := newHTTPMux(buffer, hub)
	if err != nil {
		log.Fatalf("failed to initialize embedded frontend: %v", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen on port %d: %v", *port, err)
	}

	address := listener.Addr().String()
	actualPort := *port
	if tcpAddr, ok := listener.Addr().(*net.TCPAddr); ok {
		actualPort = tcpAddr.Port
	}
	url := fmt.Sprintf("http://localhost:%d", actualPort)
	fmt.Printf("Sealog listening on %s\n", url)

	go readStdin(buffer, hub)

	if !*noOpen {
		go func() {
			if err := openBrowser(url); err != nil {
				log.Printf("failed to open browser: %v", err)
			}
		}()
	}

	server := &http.Server{
		Addr:    address,
		Handler: mux,
	}
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}
}

func newHTTPMux(buffer *RingBuffer, hub *Hub) (*http.ServeMux, error) {
	dist, err := fs.Sub(embeddedFiles, "frontend/dist")
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	fileServer := http.FileServer(http.FS(dist))
	mux.Handle("/assets/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		fileServer.ServeHTTP(w, r)
	}))
	mux.HandleFunc("/api/fields", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(buffer.Fields()); err != nil {
			log.Printf("encode fields failed: %v", err)
		}
	})
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		hub.ServeWS(w, r, buffer.Fields(), buffer.Lines())
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		index, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			http.Error(w, "frontend has not been built; run make build", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(index)
	})

	return mux, nil
}

func readStdin(buffer *RingBuffer, hub *Hub) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		fields, raw := ParseLine(scanner.Text())
		line, fieldsChanged, allFields := buffer.Add(fields, raw)
		hub.BroadcastLine(line)
		if fieldsChanged {
			hub.BroadcastFields(allFields)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("stdin read failed: %v", err)
	}
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}
