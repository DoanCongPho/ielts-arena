package middleware

import (
	"github/DoanCongPho/game-arena/internal/platform/auth"
	"github/DoanCongPho/game-arena/internal/platform/httpx"
	"net/http"
	"strings"
)

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "User does not have authorization header")
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_format", "Authorization header must be 'Bearer <token>'")
			return
		}

		claims, err := auth.VerifyToken(token)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_token", "Token is invalid or expired")
			return
		}

		user := &auth.User{
			ID:    claims.UserID,
			Email: claims.Email,
		}
		ctx := auth.SetCurrentUser(r.Context(), user)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
