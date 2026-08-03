package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// sessionCookie is the name of the signed session cookie the API sets and reads.
const sessionCookie = "air_session"

// defaultSessionLifetime is the fallback session lifetime used when a Server is
// constructed with a nil auth.Manager (see NewServer): the substitute disabled
// manager never issues tokens, so the exact value only needs to be sane.
const defaultSessionLifetime = 7 * 24 * time.Hour

// Login throttle: at most maxLoginAttempts per client IP per loginWindow. The
// fixed per-failure delay in authLogin slows a single guesser; this bounds
// concurrent and distributed attempts the delay alone cannot.
const (
	maxLoginAttempts = 10
	loginWindow      = time.Minute
)

// authExemptPaths are the exact "METHOD /path" pairs that bypass the session
// gate even when auth is enabled: the auth endpoints themselves, the health
// probe, and the Telegram webhook (which carries its own secret). The HEAD
// variants of the read-only probes are listed too: Go 1.22's mux auto-dispatches
// HEAD to the matching GET handler, so health-check HEADs must not 401.
var authExemptPaths = map[string]bool{
	"POST /auth/login":       true,
	"POST /auth/logout":      true,
	"GET /auth/status":       true,
	"HEAD /auth/status":      true,
	"POST /telegram/webhook": true,
	"GET /health":            true,
	"HEAD /health":           true,
}

// requireAuth wraps next with the session gate. When auth is disabled every
// request passes through. When enabled, exempt paths pass through and every
// other request must present a valid air_session cookie or gets a 401.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.auth.Enabled() {
			next.ServeHTTP(w, r)
			return
		}
		if authExemptPaths[r.Method+" "+r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}
		// The per-repo GitLab webhook carries the repo id in its path, so it
		// cannot be matched by the exact-path map. It is self-secured by the
		// repo's secret token (validated in the handler), so exempt it by prefix.
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/webhooks/gitlab/") {
			next.ServeHTTP(w, r)
			return
		}
		if c, err := r.Cookie(sessionCookie); err == nil && s.auth.Verify(c.Value) {
			next.ServeHTTP(w, r)
			return
		}
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	})
}

// authLogin authenticates a password and, on success, sets the session cookie.
// With auth disabled there is nothing to check, so it reports success.
func (s *Server) authLogin(w http.ResponseWriter, r *http.Request) {
	if !s.loginLimiter.allow(clientIP(r, s.trustProxy), time.Now()) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many attempts"})
		return
	}
	if !s.auth.Enabled() {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": true, "authEnabled": false})
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := decode(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if !s.auth.CheckPassword(body.Password) {
		// The compare is already constant-time; this fixed delay just blunts
		// brute-force guessing against a single-user tool.
		time.Sleep(200 * time.Millisecond)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid password"})
		return
	}
	s.setSessionCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": true})
}

// authLogout clears the session cookie.
func (s *Server) authLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsHTTPS(r, s.trustProxy),
	})
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
}

// authStatus reports whether auth is enabled and whether the caller is
// authenticated (trivially true when auth is disabled). It never requires auth.
func (s *Server) authStatus(w http.ResponseWriter, r *http.Request) {
	authenticated := !s.auth.Enabled()
	if !authenticated {
		if c, err := r.Cookie(sessionCookie); err == nil && s.auth.Verify(c.Value) {
			authenticated = true
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"authEnabled":   s.auth.Enabled(),
		"authenticated": authenticated,
	})
}

// setSessionCookie writes a fresh signed session cookie. Secure is set only when
// the request arrived over HTTPS so plain-http local development still works.
func (s *Server) setSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    s.auth.Issue(),
		Path:     "/",
		MaxAge:   int(s.auth.Lifetime().Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsHTTPS(r, s.trustProxy),
	})
}

// requestIsHTTPS reports whether the request reached us over TLS. It trusts the
// client-supplied X-Forwarded-Proto header only when the operator has opted in
// via trustProxy: otherwise a client could set X-Forwarded-Proto: https to coax
// a Secure cookie onto a plaintext connection (CWE-290).
func requestIsHTTPS(r *http.Request, trustProxy bool) bool {
	if r.TLS != nil {
		return true
	}
	return trustProxy && r.Header.Get("X-Forwarded-Proto") == "https"
}

// clientIP returns the IP used to key login rate limiting. It uses the TCP peer
// from RemoteAddr; only when trustProxy is set does it prefer the first
// X-Forwarded-For hop (an untrusted client could otherwise forge that header to
// dodge the limit or frame another IP).
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if first := strings.TrimSpace(strings.Split(xff, ",")[0]); first != "" {
				return first
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// rateLimiter is a per-key fixed-window counter safe for concurrent use. Each
// key (client IP) gets at most limit hits per window; expired windows are swept
// lazily on access so the map cannot grow without bound under IP churn.
type rateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string]*fixedWindow
}

// fixedWindow tracks one key's hit count and when its window resets.
type fixedWindow struct {
	count int
	reset time.Time
}

// newRateLimiter builds a limiter allowing limit hits per key per window.
func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, hits: make(map[string]*fixedWindow)}
}

// allow records a hit from key at now and reports whether it is within the
// limit. A fresh or fully-expired window resets to a single hit.
func (rl *rateLimiter) allow(key string, now time.Time) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Lazy cleanup: drop every window that has fully expired, bounding map size.
	for k, w := range rl.hits {
		if now.After(w.reset) {
			delete(rl.hits, k)
		}
	}

	w, ok := rl.hits[key]
	if !ok || now.After(w.reset) {
		rl.hits[key] = &fixedWindow{count: 1, reset: now.Add(rl.window)}
		return true
	}
	if w.count >= rl.limit {
		return false
	}
	w.count++
	return true
}
