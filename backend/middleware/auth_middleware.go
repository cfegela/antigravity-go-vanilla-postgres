package middleware

import (
	"context"
	"net/http"
	"strings"

	"go-vanilla-crud/auth"
)

type contextKey string

const UserContextKey contextKey = "user"

// AuthMiddleware ensures the request has a valid access token in cookie or Bearer header
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var tokenStr string

		// 1. Try reading from HttpOnly cookie
		cookie, err := r.Cookie("access_token")
		if err == nil && cookie.Value != "" {
			tokenStr = cookie.Value
		} else {
			// 2. Fallback to Authorization: Bearer <token>
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenStr == "" {
			http.Error(w, `{"error":"Unauthorized: Missing token"}`, http.StatusUnauthorized)
			return
		}

		claims, err := auth.ValidateAccessToken(tokenStr)
		if err != nil {
			http.Error(w, `{"error":"Unauthorized: `+err.Error()+`"}`, http.StatusUnauthorized)
			return
		}

		// Attach claims to context
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// GetUserFromContext retrieves authenticated claims from request context
func GetUserFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*auth.Claims)
	return claims, ok
}
