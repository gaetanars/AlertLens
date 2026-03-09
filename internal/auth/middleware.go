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
			writeAuthError(w, "admin mode is not enabled", http.StatusUnauthorized)
			return
		}
		tokenStr := ExtractBearerToken(r)
		if tokenStr == "" {
			writeAuthError(w, "missing authorization token", http.StatusUnauthorized)
			return
		}
		_, role, err := s.Validate(tokenStr)
		if err != nil {
			writeAuthError(w, "invalid or expired token", http.StatusUnauthorized)
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
				writeAuthError(w, "admin mode is not enabled", http.StatusUnauthorized)
				return
			}
			tokenStr := ExtractBearerToken(r)
			if tokenStr == "" {
				writeAuthError(w, "missing authorization token", http.StatusUnauthorized)
				return
			}
			_, role, err := s.Validate(tokenStr)
			if err != nil {
				writeAuthError(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}
			if !role.HasAtLeast(required) {
				writeAuthError(w, "insufficient permissions", http.StatusForbidden)
				return
			}
			ctx := context.WithValue(r.Context(), ctxKeyAdmin, role == RoleAdmin)
			ctx = context.WithValue(ctx, ctxKeyRole, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth populates the request context with role information when a
// valid Bearer token is present, but passes the request through even when no
// token is provided. This is suitable for read-only endpoints that are
// publicly accessible but still honour role claims for richer behaviour.
func (s *Service) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.enabled {
			tokenStr := ExtractBearerToken(r)
			if tokenStr != "" {
				if _, role, err := s.Validate(tokenStr); err == nil {
					ctx := context.WithValue(r.Context(), ctxKeyAdmin, role == RoleAdmin)
					ctx = context.WithValue(ctx, ctxKeyRole, role)
					r = r.WithContext(ctx)
				}
				// Invalid tokens on public endpoints are silently ignored —
				// the endpoint remains accessible without auth.
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequireViewer enforces the viewer role minimum on a route:
//   - When auth is disabled: allows all requests (public access).
//   - When auth is enabled: requires a valid JWT with at least viewer role.
//
// This matches the "viewer — read-only access" RBAC tier for alert/silence
// read endpoints.
func (s *Service) RequireViewer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.enabled {
			// Auth disabled → public access.
			next.ServeHTTP(w, r)
			return
		}
		tokenStr := ExtractBearerToken(r)
		if tokenStr == "" {
			writeAuthError(w, "missing authorization token", http.StatusUnauthorized)
			return
		}
		_, role, err := s.Validate(tokenStr)
		if err != nil {
			writeAuthError(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}
		if !role.HasAtLeast(RoleViewer) {
			writeAuthError(w, "insufficient permissions", http.StatusForbidden)
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyAdmin, role == RoleAdmin)
		ctx = context.WithValue(ctx, ctxKeyRole, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
