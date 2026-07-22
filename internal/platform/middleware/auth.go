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

		if claims.Type != auth.TokenTypeAccess {
			httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_token", "Refresh tokens cannot be used to authenticate requests")
			return
		}

		user := &auth.User{
			ID:    claims.UserID,
			Email: claims.Email,
			Role:  claims.Role,
		}
		ctx := auth.SetCurrentUser(r.Context(), user)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin gates a route to admin users only. It must wrap a handler
// that's already behind RequireAuth (it reads the user auth.CurrentUser
// already placed on the request context — it does not verify the token
// itself).
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.CurrentUser(r.Context())
		if !ok || !user.IsAdmin() {
			httpx.WriteError(w, http.StatusForbidden, "auth.forbidden", "This action requires an admin account")
			return
		}
		next.ServeHTTP(w, r)
	})
}
