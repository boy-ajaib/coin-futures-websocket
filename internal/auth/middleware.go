package auth

import (
	"context"
	"log/slog"
	"net/http"
)

// Context key for storing the JWT token in the request context.
type contextKey string

const (
	TokenContextKey contextKey = "jwt_token"
)

// Middleware extracts JWT from HTTP requests and stores it in the request context.
// This middleware works with Centrifuge's WebSocket upgrade flow.
type Middleware struct {
	tokenExtractor *TokenExtractor
	logger         *slog.Logger
}

// NewMiddleware creates a new auth middleware.
func NewMiddleware(logger *slog.Logger) *Middleware {
	return &Middleware{
		tokenExtractor: NewTokenExtractor(),
		logger:         logger,
	}
}

// Wrap returns an HTTP middleware that extracts JWT tokens and stores them in context.
func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header or query param
		token, err := m.tokenExtractor.Extract(
			r.Header.Get("X-Socket-Authorization"),
			r.URL.Query().Get("token"),
		)
		if err != nil {
			// Don't reject the request here - Centrifuge will handle auth
			// Just log the error for debugging
			m.logger.Debug("no JWT token found in request",
				"path", r.URL.Path,
				"error", err)
			next.ServeHTTP(w, r)
			return
		}

		// Store token in context for Centrifuge handlers to use
		ctx := WithToken(r.Context(), token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WithToken adds a JWT token to the request context.
func WithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, TokenContextKey, token)
}

// TokenFrom extracts the JWT token from the request context.
func TokenFrom(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(TokenContextKey).(string)
	return token, ok
}

// CentrifugeTransportMiddleware is a helper for Centrifuge WebSocket connections.
// It extracts JWT from the HTTP request before the WebSocket upgrade.
func CentrifugeTransportMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Centrifuge will handle the WebSocket upgrade
		// We just need to ensure the JWT is accessible
		next(w, r)
	}
}
