package websocket

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/supamanluva/ircd/internal/logger"
)

// Handler manages WebSocket connections for the IRC server
type Handler struct {
	upgrader   websocket.Upgrader
	logger     *logger.Logger
	handleConn func(net.Conn)
	origins    []string
	mu         sync.RWMutex
}

// Config holds WebSocket handler configuration
type Config struct {
	// AllowedOrigins is a list of allowed origin patterns
	// Use "*" to allow all origins (not recommended for production)
	AllowedOrigins []string
	
	// ReadBufferSize is the buffer size for reading
	ReadBufferSize int
	
	// WriteBufferSize is the buffer size for writing
	WriteBufferSize int
}

// NewHandler creates a new WebSocket handler
func NewHandler(cfg *Config, log *logger.Logger, handleConn func(net.Conn)) *Handler {
	if cfg.ReadBufferSize == 0 {
		cfg.ReadBufferSize = 1024
	}
	if cfg.WriteBufferSize == 0 {
		cfg.WriteBufferSize = 1024
	}
	if len(cfg.AllowedOrigins) == 0 {
		cfg.AllowedOrigins = []string{"*"}
	}

	h := &Handler{
		logger:     log,
		handleConn: handleConn,
		origins:    cfg.AllowedOrigins,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  cfg.ReadBufferSize,
			WriteBufferSize: cfg.WriteBufferSize,
			CheckOrigin:     nil, // Will be set below
		},
	}

	// Set CheckOrigin function
	h.upgrader.CheckOrigin = h.checkOrigin

	return h
}

// checkOrigin validates the origin header
func (h *Handler) checkOrigin(r *http.Request) bool {
	// Allow all origins if "*" is in the list
	for _, origin := range h.origins {
		if origin == "*" {
			return true
		}
	}

	// Get origin from request
	origin := r.Header.Get("Origin")
	if origin == "" {
		// No origin header, allow (for non-browser clients)
		return true
	}

	// Check against allowed origins
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	for _, allowed := range h.origins {
		if matchOrigin(origin, allowed) {
			return true
		}
	}

	h.logger.Warn("Rejected WebSocket connection from unauthorized origin", "origin", origin)
	return false
}

// matchOrigin checks if origin matches the allowed pattern
func matchOrigin(origin, pattern string) bool {
	// Exact match
	if origin == pattern {
		return true
	}

	// Wildcard pattern matching (simple implementation)
	if strings.HasPrefix(pattern, "*.") {
		domain := strings.TrimPrefix(pattern, "*.")
		return strings.HasSuffix(origin, domain)
	}

	return false
}

// ServeHTTP handles HTTP requests and upgrades them to WebSocket
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Log the connection attempt
	h.logger.Info("WebSocket connection attempt", 
		"remote", r.RemoteAddr,
		"origin", r.Header.Get("Origin"),
		"user-agent", r.Header.Get("User-Agent"))

	// Upgrade HTTP connection to WebSocket
	ws, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket connection", "error", err)
		return
	}

	// Wrap WebSocket in net.Conn interface
	conn := NewConn(ws)

	h.logger.Info("WebSocket connection established", "remote", conn.RemoteAddr())

	// Pass to IRC server handler
	// This will call the same HandleConnection method as TCP connections
	h.handleConn(conn)
}

// AddOrigin adds an allowed origin pattern at runtime
func (h *Handler) AddOrigin(origin string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.origins = append(h.origins, origin)
	h.logger.Info("Added allowed origin", "origin", origin)
}

// RemoveOrigin removes an allowed origin pattern
func (h *Handler) RemoveOrigin(origin string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	for i, o := range h.origins {
		if o == origin {
			h.origins = append(h.origins[:i], h.origins[i+1:]...)
			h.logger.Info("Removed allowed origin", "origin", origin)
			return
		}
	}
}

// GetOrigins returns the list of allowed origins
func (h *Handler) GetOrigins() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	origins := make([]string, len(h.origins))
	copy(origins, h.origins)
	return origins
}

// HealthCheck returns an HTTP handler for health checks
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","service":"ircd-websocket"}`)
}
