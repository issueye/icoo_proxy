package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"icoo_proxy/internal/config"
	"icoo_proxy/internal/provider"
)

// Server is the AI gateway HTTP server.
type Server struct {
	server  *http.Server
	host    string
	port    int
	running bool
	handler *Handler
}

var (
	instance *Server
)

// GetServer returns the singleton Server instance.
func GetServer() *Server {
	if instance == nil {
		instance = &Server{
			handler: NewHandler(),
		}
	}
	return instance
}

// Start starts the gateway HTTP server.
func (s *Server) Start(host string, port int) error {
	host = strings.TrimSpace(host)
	if host == "" {
		host = "127.0.0.1"
	}
	addr := fmt.Sprintf("%s:%d", host, port)

	if s.running {
		return fmt.Errorf("gateway already running on %s:%d", s.host, s.port)
	}

	s.host = host
	s.port = port

	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/v1/chat/completions", s.handler.ChatCompletions)
	mux.HandleFunc("/v1/responses", s.handler.Responses)
	mux.HandleFunc("/v1/models", s.handler.Models)
	mux.HandleFunc("/v1/health", s.handler.Health)

	// Apply middleware chain
	var handler http.Handler = mux
	handler = authMiddleware(handler)
	handler = corsMiddleware(handler)
	handler = recoveryMiddleware(handler)
	handler = loggingMiddleware(handler)

	s.server = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	go func() {
		log.Printf("[Gateway] Starting on %s", addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Gateway] Server error: %v", err)
		}
	}()

	s.running = true
	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)
	return nil
}

// Stop gracefully stops the gateway server.
func (s *Server) Stop() error {
	if !s.running || s.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("gateway shutdown error: %w", err)
	}

	s.running = false
	log.Println("[Gateway] Stopped")
	return nil
}

// IsRunning returns whether the gateway is currently running.
func (s *Server) IsRunning() bool {
	return s.running
}

// GetHost returns the host the gateway is listening on.
func (s *Server) GetHost() string {
	if strings.TrimSpace(s.host) == "" {
		return "127.0.0.1"
	}
	return s.host
}

// GetPort returns the port the gateway is listening on.
func (s *Server) GetPort() int {
	return s.port
}

// --- Middleware ---

type contextKey string

const apiKeyContextKey contextKey = "api-key"

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-api-key, anthropic-version")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[Gateway] Panic recovered: %v", err)
				http.Error(w, `{"error":{"message":"Internal server error","type":"internal_error"}}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := r.URL.Path

		// Wrap ResponseWriter to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		log.Printf("[Gateway] %s %s %d %s", r.Method, path, wrapped.statusCode, duration.Round(time.Millisecond))
	})
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKeys := provider.GetManager().GetAPIKeys()
		if len(apiKeys) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		providedKey := strings.TrimSpace(r.Header.Get("x-api-key"))
		if providedKey == "" {
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				providedKey = strings.TrimSpace(authHeader[7:])
			}
		}

		if !isAuthorizedAPIKey(apiKeys, providedKey) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"Unauthorized","type":"authentication_error"}}`))
			return
		}

		if providedKey != "" {
			r = r.WithContext(context.WithValue(r.Context(), apiKeyContextKey, providedKey))
		}

		next.ServeHTTP(w, r)
	})
}

func isAuthorizedAPIKey(apiKeys []config.ApiKeyConfig, providedKey string) bool {
	providedKey = strings.TrimSpace(providedKey)
	if providedKey == "" {
		return false
	}
	for _, item := range apiKeys {
		if !item.Enabled {
			continue
		}
		if strings.TrimSpace(item.Key) == providedKey {
			return true
		}
	}
	return false
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// IsStreamingRequest checks if the request should be streamed.
func IsStreamingRequest(body []byte) bool {
	return strings.Contains(string(body), `"stream":true`) || strings.Contains(string(body), `"stream": true`)
}
