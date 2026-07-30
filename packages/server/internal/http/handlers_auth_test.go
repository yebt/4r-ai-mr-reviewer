package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
	"time"
)

// postJSONWith posts a JSON body using the given client, so cookie-jar state is
// carried across login/logout/protected-route calls.
func postJSONWith(t *testing.T, client *http.Client, url string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := client.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

// jarClient is an HTTP client that stores cookies, so a login response's
// Set-Cookie is replayed on later requests to the same test server.
func jarClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	return &http.Client{Jar: jar}
}

// TestAuthDisabledPassthrough proves that with no AIR_AUTH_PASSWORD every route
// is reachable without a cookie, exactly as before this feature.
func TestAuthDisabledPassthrough(t *testing.T) {
	srv := newTestServer(t) // auth disabled

	resp, err := http.Get(srv.URL + "/accounts")
	if err != nil {
		t.Fatalf("GET /accounts: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("protected route with auth disabled = %d, want 200", resp.StatusCode)
	}

	// /auth/login is a no-op success when auth is disabled.
	loginResp := postJSON(t, srv.URL+"/auth/login", map[string]any{"password": "anything"})
	defer loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login with auth disabled = %d, want 200", loginResp.StatusCode)
	}
	var body struct {
		Authenticated bool `json:"authenticated"`
		AuthEnabled   bool `json:"authEnabled"`
	}
	decodeBody(t, loginResp, &body)
	if !body.Authenticated || body.AuthEnabled {
		t.Fatalf("login (disabled) body = %+v, want authenticated=true authEnabled=false", body)
	}
}

// TestProtectedRouteRejectsWithoutValidCookie covers the 401 paths when auth is
// enabled: no cookie and a garbage cookie.
func TestProtectedRouteRejectsWithoutValidCookie(t *testing.T) {
	srv := newTestServerWithAuth(t, "hunter2")

	resp, err := http.Get(srv.URL + "/accounts")
	if err != nil {
		t.Fatalf("GET /accounts: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("protected route without cookie = %d, want 401", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/accounts", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "tampered.value"})
	bad, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /accounts (bad cookie): %v", err)
	}
	bad.Body.Close()
	if bad.StatusCode != http.StatusUnauthorized {
		t.Fatalf("protected route with invalid cookie = %d, want 401", bad.StatusCode)
	}
}

// TestExemptPathsReachableWithoutCookie confirms the exempt set bypasses the
// gate when auth is enabled.
func TestExemptPathsReachableWithoutCookie(t *testing.T) {
	srv := newTestServerWithAuth(t, "hunter2")

	for _, path := range []string{"/health", "/auth/status"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("exempt %s without cookie = %d, want 200", path, resp.StatusCode)
		}
	}

	// The Telegram webhook is exempt (self-secured); dormant here → 200.
	wh := postWebhook(t, srv.URL+"/telegram/webhook", "", validReposUpdate)
	wh.Body.Close()
	if wh.StatusCode != http.StatusOK {
		t.Fatalf("exempt /telegram/webhook without cookie = %d, want 200", wh.StatusCode)
	}
}

// TestLoginLogoutFlow exercises the full enabled-auth cycle: wrong password is
// rejected, correct password sets a cookie that unlocks protected routes, and
// logout clears it so access is 401 again.
func TestLoginLogoutFlow(t *testing.T) {
	srv := newTestServerWithAuth(t, "hunter2")
	client := jarClient(t)

	// Wrong password → 401, no cookie stored.
	bad := postJSONWith(t, client, srv.URL+"/auth/login", map[string]any{"password": "wrong"})
	bad.Body.Close()
	if bad.StatusCode != http.StatusUnauthorized {
		t.Fatalf("login wrong password = %d, want 401", bad.StatusCode)
	}

	// Correct password → 200 and a session cookie.
	ok := postJSONWith(t, client, srv.URL+"/auth/login", map[string]any{"password": "hunter2"})
	if ok.StatusCode != http.StatusOK {
		t.Fatalf("login correct password = %d, want 200", ok.StatusCode)
	}
	ok.Body.Close()
	if !hasSessionCookie(client, srv.URL) {
		t.Fatal("successful login did not set the session cookie")
	}

	// The stored cookie unlocks a protected route.
	acc, err := client.Get(srv.URL + "/accounts")
	if err != nil {
		t.Fatalf("GET /accounts (authed): %v", err)
	}
	acc.Body.Close()
	if acc.StatusCode != http.StatusOK {
		t.Fatalf("protected route with valid cookie = %d, want 200", acc.StatusCode)
	}

	// Logout clears the cookie; access is 401 again.
	out := postJSONWith(t, client, srv.URL+"/auth/logout", nil)
	out.Body.Close()
	if out.StatusCode != http.StatusOK {
		t.Fatalf("logout = %d, want 200", out.StatusCode)
	}
	after, err := client.Get(srv.URL + "/accounts")
	if err != nil {
		t.Fatalf("GET /accounts (after logout): %v", err)
	}
	after.Body.Close()
	if after.StatusCode != http.StatusUnauthorized {
		t.Fatalf("protected route after logout = %d, want 401", after.StatusCode)
	}
}

// TestAuthStatus pins the /auth/status contract in both modes.
func TestAuthStatus(t *testing.T) {
	// Disabled: authEnabled=false, authenticated=true (nothing to check).
	disabled := newTestServer(t)
	dResp, _ := http.Get(disabled.URL + "/auth/status")
	var d struct {
		AuthEnabled   bool `json:"authEnabled"`
		Authenticated bool `json:"authenticated"`
	}
	decodeBody(t, dResp, &d)
	if d.AuthEnabled || !d.Authenticated {
		t.Fatalf("status (disabled) = %+v, want authEnabled=false authenticated=true", d)
	}

	// Enabled, no cookie: authEnabled=true, authenticated=false.
	enabled := newTestServerWithAuth(t, "hunter2")
	eResp, _ := http.Get(enabled.URL + "/auth/status")
	var e struct {
		AuthEnabled   bool `json:"authEnabled"`
		Authenticated bool `json:"authenticated"`
	}
	decodeBody(t, eResp, &e)
	if !e.AuthEnabled || e.Authenticated {
		t.Fatalf("status (enabled, no cookie) = %+v, want authEnabled=true authenticated=false", e)
	}
}

// TestLoginRateLimited proves the per-IP login throttle: maxLoginAttempts rapid
// attempts are allowed, and the next one is refused with 429 before the password
// is even checked. Correct credentials are used so the fixed wrong-password delay
// never runs and the test stays fast.
func TestLoginRateLimited(t *testing.T) {
	srv := newTestServerWithAuth(t, "hunter2")

	for i := 0; i < maxLoginAttempts; i++ {
		resp := postJSON(t, srv.URL+"/auth/login", map[string]any{"password": "hunter2"})
		resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			t.Fatalf("attempt %d = 429 before the limit was reached", i+1)
		}
	}

	over := postJSON(t, srv.URL+"/auth/login", map[string]any{"password": "hunter2"})
	defer over.Body.Close()
	if over.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("attempt %d = %d, want 429", maxLoginAttempts+1, over.StatusCode)
	}
	var body struct {
		Error string `json:"error"`
	}
	decodeBody(t, over, &body)
	if body.Error != "too many attempts" {
		t.Fatalf("429 body error = %q, want %q", body.Error, "too many attempts")
	}
}

// TestTamperedCookiePayloadRejected logs in, extracts the real air_session
// cookie, flips one byte of its payload segment, and confirms the signature
// mismatch makes the replayed request 401.
func TestTamperedCookiePayloadRejected(t *testing.T) {
	srv := newTestServerWithAuth(t, "hunter2")
	client := jarClient(t)

	ok := postJSONWith(t, client, srv.URL+"/auth/login", map[string]any{"password": "hunter2"})
	ok.Body.Close()

	raw := sessionCookieValue(client, srv.URL)
	if raw == "" {
		t.Fatal("login did not set a session cookie")
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		t.Fatalf("cookie value = %q, want two dot-separated segments", raw)
	}
	// Flip one payload byte to another valid base64url char: it still decodes,
	// but the HMAC over the changed payload no longer matches parts[1].
	tampered := flipFirstChar(parts[0]) + "." + parts[1]

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/accounts", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: tampered})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /accounts (tampered cookie): %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("tampered-payload cookie = %d, want 401", resp.StatusCode)
	}
}

// TestExpiredCookieRejected builds a server whose sessions have a negative
// lifetime, so a freshly-issued cookie is already expired. The Set-Cookie is read
// straight off the login response (a cookie jar would discard the negative
// Max-Age) and replayed, which must 401 — no sleeping.
func TestExpiredCookieRejected(t *testing.T) {
	srv := newTestServerWithAuthLifetime(t, "hunter2", -time.Minute)

	login := postJSON(t, srv.URL+"/auth/login", map[string]any{"password": "hunter2"})
	defer login.Body.Close()
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login = %d, want 200", login.StatusCode)
	}
	var value string
	for _, c := range login.Cookies() {
		if c.Name == sessionCookie {
			value = c.Value
		}
	}
	if value == "" {
		t.Fatal("login did not set a session cookie")
	}

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/accounts", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: value})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /accounts (expired cookie): %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expired cookie = %d, want 401", resp.StatusCode)
	}
}

// sessionCookieValue returns the raw session cookie value from the client's jar,
// or "" if absent.
func sessionCookieValue(client *http.Client, rawURL string) string {
	req, _ := http.NewRequest(http.MethodGet, rawURL, nil)
	for _, c := range client.Jar.Cookies(req.URL) {
		if c.Name == sessionCookie {
			return c.Value
		}
	}
	return ""
}

// flipFirstChar swaps the first character for a different base64url character so
// the segment still decodes but its bytes differ.
func flipFirstChar(s string) string {
	if s == "" {
		return "A"
	}
	b := []byte(s)
	if b[0] == 'A' {
		b[0] = 'B'
	} else {
		b[0] = 'A'
	}
	return string(b)
}

// hasSessionCookie reports whether the client's jar holds the session cookie for
// the server URL.
func hasSessionCookie(client *http.Client, rawURL string) bool {
	req, _ := http.NewRequest(http.MethodGet, rawURL, nil)
	for _, c := range client.Jar.Cookies(req.URL) {
		if c.Name == sessionCookie && c.Value != "" {
			return true
		}
	}
	return false
}
