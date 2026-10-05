package proxy

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Hub fans a single upstream SSE stream out to many console clients.
type Hub struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: map[chan string]struct{}{}}
}

func (h *Hub) subscribe() chan string {
	ch := make(chan string, 64)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *Hub) unsubscribe(ch chan string) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
}

func (h *Hub) broadcast(line string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- line:
		default: // slow client: drop the frame rather than block the stream
		}
	}
}

// Run connects to the upstream stream and reconnects with backoff.
func (h *Hub) Run(ctx context.Context, streamURL string) {
	backoff := time.Second
	for ctx.Err() == nil {
		if err := h.consume(ctx, streamURL); err == nil {
			backoff = time.Second
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (h *Hub) consume(ctx context.Context, streamURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		if line := sc.Text(); len(line) > 5 && line[:5] == "data:" {
			h.broadcast(line)
		}
	}
	return sc.Err()
}

// Handler streams frames to one client with periodic heartbeats.
func (h *Hub) Handler(heartbeat time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fl, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		ch := h.subscribe()
		defer h.unsubscribe(ch)
		t := time.NewTicker(heartbeat)
		defer t.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case line := <-ch:
				fmt.Fprintf(w, "%s\n\n", line)
			case <-t.C:
				fmt.Fprint(w, ": keepalive\n\n")
			}
			fl.Flush()
		}
	}
}
