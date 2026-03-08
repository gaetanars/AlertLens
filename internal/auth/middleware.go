package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const (
	ctxKeyAdmin contextKey = "admin"
	ctxKeyRole  contextKey = "role"
)

// Middleware returns an HTTP middleware that validates the Bearer JWT.
// On success it stores both the legacy "admin" boolean and the granular Role
// in the request context, then forwards to the next handler.
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
		_, role, err := s.Validate(tokenStr)
		if err != nil {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyAdmin, true)
		ctx = context.WithValue(ctx, ctxKeyRole, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole returns a middleware that only allows requests whose JWT carries
// at least the specified role level.  It must be chained after Middleware (which
// performs the actual JWT validation and stores the role in context).
func (s *Service) RequireRole(required Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
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
			_, role, err := s.Validate(tokenStr)
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}
			if !role.HasAtLeast(required) {
				http.Error(w, `{"error":"insufficient permissions"}`, http.StatusForbidden)
				return
			}
			ctx := context.WithValue(r.Context(), ctxKeyAdmin, role == RoleAdmin)
			ctx = context.WithValue(ctx, ctxKeyRole, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetRole returns the Role stored in the request context.
// Returns an empty string if no role is present (unauthenticated request).
func GetRole(r *http.Request) Role {
	v, _ := r.Context().Value(ctxKeyRole).(Role)
	return v
}

// IsAdmin returns true if the request context carries an admin-role token.
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
