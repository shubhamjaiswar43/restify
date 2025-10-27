package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/shubhamjaiswar43/restify/internal/helper"
)

// AuthMiddleware checks JWT and user roles
func NewAuthMiddleware(secret string) func(allowedRoles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(allowedRoles ...string) func(http.HandlerFunc) http.HandlerFunc {
		return func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					slog.Warn("missing Authorization header")
					helper.WriteSimpleError(w, http.StatusUnauthorized, "Missing Authorization header")
					return
				}

				tokenParts := strings.Split(authHeader, " ")
				if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
					slog.Warn("invalid Authorization header format", slog.String("header", authHeader))
					helper.WriteSimpleError(w, http.StatusUnauthorized, "Invalid Authorization header format")
					return
				}

				tokenStr := tokenParts[1]
				jwtManager := NewJWTManager(secret, 24*time.Hour)

				claims, err := jwtManager.Verify(tokenStr)
				if err != nil {
					slog.Error("invalid or expired token", slog.String("error", err.Error()))
					helper.WriteSimpleError(w, http.StatusUnauthorized, "Invalid or expired token")
					return
				}

				// Check allowed roles
				allowed := false
				for _, role := range allowedRoles {
					if claims.Role == role {
						allowed = true
						break
					}
				}
				if !allowed {
					slog.Warn("access denied", slog.String("user_role", claims.Role), slog.Any("allowed_roles", allowedRoles))
					helper.WriteSimpleError(w, http.StatusForbidden, "Access denied: insufficient permissions")
					return
				}

				// Add claims to request context
				ctx := context.WithValue(r.Context(), "claims", claims)
				slog.Info("authenticated request",
					slog.String("user_id", claims.UserID),
					slog.String("role", claims.Role),
					slog.String("path", r.URL.Path),
				)

				next(w, r.WithContext(ctx))
			}
		}
	}
}
