package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const ctxKeyAdmin contextKey = "admin"

// Middleware returns an HTTP middleware that validates the Bearer JWT.
// If the token is valid, it sets the "admin" key in the request context.
// Requests without a valid token are rejected with 401.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.enabled {
			http.Error(w, `{"error":"admin mode is not enabled"}`, http.StatusUnauthorized)
			return
		}
		tokenStr := ExtractBearerToken(r)
		if tokenStr == "" {
			http.Error(w, `{"error":"missing authorization token"}`, http.StatusUnauthorized)
			return
		}
		if _, err := s.Validate(tokenStr); err != nil {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyAdmin, true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// IsAdmin returns true if the request context carries a valid admin token.
func IsAdmin(r *http.Request) bool {
	v, _ := r.Context().Value(ctxKeyAdmin).(bool)
	return v
}

// ExtractBearerToken extracts the Bearer token from the Authorization header.
// Exported so handler packages can reuse it without duplicating the logic.
func ExtractBearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(header, "Bearer ")
}
