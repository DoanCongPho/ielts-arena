package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const allowedOrigin = "https://ielts-arena.example"

// This is the bug the rewrite exists to prevent: an unconfigured
// deployment used to install a middleware that rejected every origin,
// blocking the whole API while the server logs looked clean. Now it
// installs nothing at all.
func TestNewCORSReturnsNilWhenNoOriginsConfigured(t *testing.T) {
	for _, origins := range [][]string{nil, {}, {""}, {"  ", ""}} {
		if got := NewCORS(origins); got != nil {
			t.Errorf("NewCORS(%q) = %v, want nil so main wraps nothing", origins, got)
		}
	}
}

func TestNewCORSBuildsWhenOriginsConfigured(t *testing.T) {
	if NewCORS([]string{allowedOrigin}) == nil {
		t.Error("NewCORS should build a middleware when an origin is configured")
	}
	if NewCORS([]string{"*"}) == nil {
		t.Error("NewCORS should build a middleware for a wildcard origin")
	}
}

func newCORSRecorder(t *testing.T, c *CORS, method, origin string, preflight bool) (*httptest.ResponseRecorder, bool) {
	t.Helper()

	var called bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(method, "/api/tests", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if preflight {
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	}

	rec := httptest.NewRecorder()
	c.Handler(next).ServeHTTP(rec, req)
	return rec, called
}

func TestCORSAllowedOriginGetsHeader(t *testing.T) {
	c := NewCORS([]string{allowedOrigin})

	rec, called := newCORSRecorder(t, c, http.MethodGet, allowedOrigin, false)

	if !called {
		t.Error("an allowed origin's request should reach the handler")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != allowedOrigin {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, allowedOrigin)
	}
	// Without Vary, a shared cache can serve one origin's Allow-Origin
	// header to a different origin.
	if got := rec.Header().Get("Vary"); !strings.Contains(got, "Origin") {
		t.Errorf("Vary = %q, want it to include Origin", got)
	}
}

func TestCORSDisallowedOriginGetsNoHeader(t *testing.T) {
	c := NewCORS([]string{allowedOrigin})

	rec, called := newCORSRecorder(t, c, http.MethodGet, "https://evil.example", false)

	// The request still runs — the browser is what refuses to hand the
	// response to the caller's JavaScript.
	if !called {
		t.Error("a non-preflight request should still reach the handler")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want it absent for an unknown origin", got)
	}
}

func TestCORSAllowedPreflight(t *testing.T) {
	c := NewCORS([]string{allowedOrigin})

	rec, called := newCORSRecorder(t, c, http.MethodOptions, allowedOrigin, true)

	if called {
		t.Error("a preflight must be answered by the middleware, not passed to the handler")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != allowedOrigin {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, allowedOrigin)
	}
	for _, h := range []string{"Access-Control-Allow-Methods", "Access-Control-Allow-Headers"} {
		if rec.Header().Get(h) == "" {
			t.Errorf("%s should be set on an accepted preflight", h)
		}
	}
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Error("Allow-Headers must include Authorization — every API call sends it")
	}
}

// The old version answered a rejected preflight with 204, so a
// misconfiguration was invisible server-side. It must be a visible 403.
func TestCORSRejectedPreflightIsForbiddenNot204(t *testing.T) {
	c := NewCORS([]string{allowedOrigin})

	rec, called := newCORSRecorder(t, c, http.MethodOptions, "https://evil.example", true)

	if called {
		t.Error("a rejected preflight must not reach the handler")
	}
	if rec.Code == http.StatusNoContent {
		t.Fatal("a rejected preflight answered 204 — that is the bug this rewrite fixes")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want it absent", got)
	}
}

// curl, health checks and server-to-server calls send no Origin. They are
// not browser requests, so there is nothing to negotiate.
func TestCORSRequestWithoutOriginPassesThroughUntouched(t *testing.T) {
	c := NewCORS([]string{allowedOrigin})

	rec, called := newCORSRecorder(t, c, http.MethodGet, "", false)

	if !called {
		t.Error("a request with no Origin should reach the handler")
	}
	if got := rec.Header().Get("Vary"); got != "" {
		t.Errorf("Vary = %q, want nothing added for a non-CORS request", got)
	}
}

func TestCORSWildcardAllowsAnyOrigin(t *testing.T) {
	c := NewCORS([]string{"*"})

	rec, _ := newCORSRecorder(t, c, http.MethodGet, "https://anything.example", false)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://anything.example" {
		t.Errorf("Access-Control-Allow-Origin = %q, want the request's origin echoed back", got)
	}
}

func TestCORSTrimsWhitespaceInConfiguredOrigins(t *testing.T) {
	c := NewCORS([]string{"  " + allowedOrigin + "  "})

	rec, _ := newCORSRecorder(t, c, http.MethodGet, allowedOrigin, false)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != allowedOrigin {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q — ALLOWED_ORIGINS is comma-separated and often has spaces", got, allowedOrigin)
	}
}
