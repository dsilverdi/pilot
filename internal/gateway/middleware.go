package gateway

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dsilverdi/pilot/internal/gateway/apikey"
)

// LoggingMiddleware logs all incoming requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		log.Printf("%s %s %d %s",
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			time.Since(start),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// CORSMiddleware adds CORS headers
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, o := range allowedOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}

			if allowed && origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			// Handle preflight
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AuthMiddleware validates API key using the apikey.Manager
func AuthMiddleware(keyManager *apikey.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth if no API keys are configured
			if keyManager == nil || !keyManager.HasKeys() {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for health endpoint
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			// Check API key from header
			providedKey := r.Header.Get("X-API-Key")
			if providedKey == "" {
				// Also check Authorization header
				auth := r.Header.Get("Authorization")
				if after, found := strings.CutPrefix(auth, "Bearer "); found {
					providedKey = after
				}
			}

			if providedKey == "" {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error": "API key required", "code": 401}`, http.StatusUnauthorized)
				return
			}

			valid, _, err := keyManager.Validate(providedKey)
			if err != nil {
				log.Printf("API key validation error: %v", err)
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error": "internal error", "code": 500}`, http.StatusInternalServerError)
				return
			}

			if !valid {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error": "invalid API key", "code": 401}`, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ChainMiddleware chains multiple middleware functions
func ChainMiddleware(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
