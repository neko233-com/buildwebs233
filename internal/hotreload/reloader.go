package hotreload

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Event struct {
	Type      string    `json:"type"`
	File      string    `json:"file,omitempty"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type Hub struct {
	mu      sync.Mutex
	clients map[chan Event]struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[chan Event]struct{}),
	}
}

func (h *Hub) Register() chan Event {
	ch := make(chan Event, 4)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unregister(ch chan Event) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
	close(ch)
}

func (h *Hub) Broadcast(event Event) {
	event.Timestamp = time.Now()
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- event:
		default:
			// drop message to avoid blocking clients
		}
	}
}

func (h *Hub) ServeSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, "retry: 1000\n\n"); err != nil {
		log.Printf("[reload] sse init write failed: %v", err)
		return
	}
	flusher.Flush()

	ch := h.Register()
	defer h.Unregister(ch)

	for {
		select {
		case <-r.Context().Done():
			return
		case e := <-ch:
			payload, _ := json.Marshal(e)
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", sanitizeEvent(e.Type), payload); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func sanitizeEvent(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return "reload"
	}
	return name
}

func ReloadClientScript() string {
	return `(() => {
  const source = new EventSource('/api/reload');
  source.addEventListener('config', function() {
    window.location.reload();
  });
  source.addEventListener('html', function() {
    window.location.reload();
  });
  source.addEventListener('error', function() {
    console.warn('[hotreload] connection lost, retrying');
  });
})();`
}

