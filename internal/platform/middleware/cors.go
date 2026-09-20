package middleware

import (
	"net/http"
	"strings"
)

// CORS grants named browser origins permission to read this API's
// responses. It is only needed when the browser talks to the API on a
// different origin than it loaded the page from — split hosting, e.g. the
// SPA on Vercel and this API on another domain.
//
// The default deployment does NOT need it: nginx serves the SPA and
// proxies /api to this service, so every request is same-origin and the
// browser's same-origin policy never comes into play. NewCORS returns nil
// for that case and main wraps nothing, rather than installing a
// middleware that silently rejects every origin.
type CORS struct {
	// allowed is the exact-match set. Empty when allowAll is set.
	allowed map[string]struct{}
	// allowAll mirrors a configured "*".
	allowAll bool
	methods  string
	headers  string
	maxAge   string
}

// NewCORS builds the middleware from a configured origin list, or returns
// nil when the list is empty.
//
// Returning nil rather than a deny-everything middleware is the whole
// point: the previous version treated "no origins configured" as "reject
// every origin", which blocked the entire API with no server-side error —
// the browser failed the preflight while the Go logs showed a clean 204.
// Now an unconfigured deployment is simply same-origin, and a misconfigured
// one is visible because rejected preflights answer 403.
func NewCORS(origins []string) *CORS {
	c := &CORS{
		allowed: make(map[string]struct{}, len(origins)),
		methods: "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		headers: "Content-Type, Authorization",
		maxAge:  "600",
	}
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" {
			c.allowAll = true
			continue
		}
		c.allowed[o] = struct{}{}
	}
	if !c.allowAll && len(c.allowed) == 0 {
		return nil
	}
	return c
}

func (c *CORS) isAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	if c.allowAll {
		return true
	}
	_, ok := c.allowed[origin]
	return ok
}

// Handler wraps next with CORS negotiation.
func (c *CORS) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// No Origin header: not a cross-origin browser request at all
		// (curl, a health check, server-to-server). Nothing to negotiate.
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		// The response body/headers now depend on which origin asked, so
		// any cache in front of this must key on Origin too. Without this,
		// a shared cache can hand origin A's Allow-Origin header to
		// origin B and break it intermittently.
		w.Header().Add("Vary", "Origin")

		allowed := c.isAllowed(origin)
		preflight := r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""

		if !allowed {
			if preflight {
				// Answer a rejected preflight with 403, not the 204 the
				// old version sent. The browser blocks it either way, but
				// a 403 makes the rejection visible in server logs and
				// access logs instead of looking like a success.
				http.Error(w, "origin not allowed", http.StatusForbidden)
				return
			}
			// A non-preflight request from an unknown origin still runs:
			// the browser simply won't let the caller's JS read the
			// response, which is the same-origin policy doing its job.
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)

		if preflight {
			w.Header().Set("Access-Control-Allow-Methods", c.methods)
			w.Header().Set("Access-Control-Allow-Headers", c.headers)
			w.Header().Set("Access-Control-Max-Age", c.maxAge)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
