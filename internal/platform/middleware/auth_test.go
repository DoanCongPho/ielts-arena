package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github/DoanCongPho/game-arena/internal/platform/auth"
)

const testSecret = "middleware-test-secret-32-chars-min!"

// okHandler records whether the middleware let the request through, and
// what user it put on the context.
func okHandler(called *bool, seen **auth.User) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		if u, ok := auth.CurrentUser(r.Context()); ok {
			*seen = u
		}
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireAuth(t *testing.T) {
	auth.Init([]byte(testSecret))

	user := &auth.User{ID: 7, Email: "learner@example.com", Role: auth.RoleUser}
	access, err := auth.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	refresh, err := auth.GenerateRefreshToken(user)
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}

	tests := []struct {
		name       string
		header     string
		wantStatus int
		wantCalled bool
	}{
		{"no Authorization header", "", http.StatusUnauthorized, false},
		{"missing Bearer prefix", access, http.StatusUnauthorized, false},
		{"wrong scheme", "Basic " + access, http.StatusUnauthorized, false},
		{"malformed token", "Bearer not-a-jwt", http.StatusUnauthorized, false},
		// A refresh token is a valid signature but the wrong kind of
		// credential — accepting it would defeat the short access TTL.
		{"refresh token used as credential", "Bearer " + refresh, http.StatusUnauthorized, false},
		{"valid access token", "Bearer " + access, http.StatusOK, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var called bool
			var seen *auth.User

			req := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()

			RequireAuth(okHandler(&called, &seen)).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if called != tc.wantCalled {
				t.Errorf("next handler called = %v, want %v", called, tc.wantCalled)
			}
			if tc.wantCalled {
				if seen == nil {
					t.Fatal("expected the authenticated user on the request context")
				}
				if seen.ID != 7 {
					t.Errorf("context user ID = %d, want 7", seen.ID)
				}
				if seen.Role != auth.RoleUser {
					t.Errorf("context user Role = %q, want %q", seen.Role, auth.RoleUser)
				}
			}
		})
	}
}

func TestRequireAdmin(t *testing.T) {
	tests := []struct {
		name       string
		user       *auth.User
		setUser    bool
		wantStatus int
		wantCalled bool
	}{
		// RequireAdmin reads the user RequireAuth put on the context. If
		// it is ever mounted without RequireAuth in front, it must deny
		// rather than wave the request through.
		{"no user on context", nil, false, http.StatusForbidden, false},
		{"regular user", &auth.User{ID: 1, Role: auth.RoleUser}, true, http.StatusForbidden, false},
		{"empty role", &auth.User{ID: 1, Role: ""}, true, http.StatusForbidden, false},
		{"admin user", &auth.User{ID: 1, Role: auth.RoleAdmin}, true, http.StatusOK, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var called bool
			var seen *auth.User

			req := httptest.NewRequest(http.MethodPost, "/api/tests", nil)
			if tc.setUser {
				req = req.WithContext(auth.SetCurrentUser(req.Context(), tc.user))
			}
			rec := httptest.NewRecorder()

			RequireAdmin(okHandler(&called, &seen)).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if called != tc.wantCalled {
				t.Errorf("next handler called = %v, want %v", called, tc.wantCalled)
			}
		})
	}
}
