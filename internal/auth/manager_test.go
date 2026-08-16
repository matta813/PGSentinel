package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionLifecycle(t *testing.T) {
	m, err := New(Config{Password: "correct horse battery staple", SessionTTL: time.Hour, SecureCookies: true})
	if err != nil {
		t.Fatal(err)
	}
	if !m.CheckPassword("correct horse battery staple") || m.CheckPassword("wrong") {
		t.Fatal("password comparison failed")
	}
	w := httptest.NewRecorder()
	if err := m.Start(w); err != nil {
		t.Fatal(err)
	}
	result := w.Result()
	if len(result.Cookies()) != 1 || !result.Cookies()[0].HttpOnly || !result.Cookies()[0].Secure || result.Cookies()[0].SameSite != http.SameSiteStrictMode {
		t.Fatal("session cookie is not hardened")
	}
	r := httptest.NewRequest("GET", "/api/v1/servers", nil)
	r.AddCookie(result.Cookies()[0])
	if !m.Valid(r) {
		t.Fatal("new session is not valid")
	}
	m.End(httptest.NewRecorder(), r)
	if m.Valid(r) {
		t.Fatal("ended session remains valid")
	}
}

func TestLoginAttemptLimit(t *testing.T) {
	m, err := New(Config{Password: "a sufficiently long password", MaxAttempts: 2, AttemptWindow: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	r := requestFrom("192.0.2.10:1234", "")
	if !m.AllowAttempt(r) || !m.AllowAttempt(r) || m.AllowAttempt(r) {
		t.Fatal("attempt limit was not enforced per client address")
	}
	m.ResetAttempts(r)
	if !m.AllowAttempt(r) {
		t.Fatal("successful login did not reset attempt limit")
	}
}

func TestExpiredSessionsAndAttemptsArePruned(t *testing.T) {
	m, err := New(Config{Password: "a sufficiently long password", SessionTTL: time.Second, MaxAttempts: 1, AttemptWindow: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	m.now = func() time.Time { return now }
	w := httptest.NewRecorder()
	if err := m.Start(w); err != nil {
		t.Fatal(err)
	}
	r := requestFrom("192.0.2.10:1234", "")
	r.AddCookie(w.Result().Cookies()[0])
	if !m.Valid(r) || !m.AllowAttempt(r) || m.AllowAttempt(r) {
		t.Fatal("initial expiry state is incorrect")
	}
	now = now.Add(2 * time.Second)
	if m.Valid(r) {
		t.Fatal("expired session remains valid")
	}
	if !m.AllowAttempt(r) {
		t.Fatal("expired attempt window was not pruned")
	}
}

func TestTrustedProxyUsesForwardedClientAndIgnoresUntrustedHeader(t *testing.T) {
	m, err := New(Config{Password: "a sufficiently long password", MaxAttempts: 1, TrustedProxies: []string{"10.0.0.0/8"}})
	if err != nil {
		t.Fatal(err)
	}
	proxied := requestFrom("10.0.0.2:443", "198.51.100.7, 10.0.0.3")
	if !m.AllowAttempt(proxied) || m.AllowAttempt(proxied) {
		t.Fatal("forwarded client was not rate limited")
	}
	other := requestFrom("10.0.0.2:443", "198.51.100.8")
	if !m.AllowAttempt(other) {
		t.Fatal("different forwarded client shared a limit")
	}
	untrusted := requestFrom("203.0.113.9:443", "198.51.100.8")
	if !m.AllowAttempt(untrusted) || m.AllowAttempt(untrusted) {
		t.Fatal("untrusted forwarded header was accepted")
	}
}

func TestAttemptTrackingIsBounded(t *testing.T) {
	m, err := New(Config{Password: "a sufficiently long password", MaxAttempts: 1, MaxClients: 2})
	if err != nil {
		t.Fatal(err)
	}
	for _, address := range []string{"192.0.2.1:1", "192.0.2.2:1", "192.0.2.3:1", "192.0.2.4:1"} {
		m.AllowAttempt(requestFrom(address, ""))
	}
	if len(m.attempts) > 3 {
		t.Fatalf("attempt map grew beyond bounded clients: %d", len(m.attempts))
	}
}

func TestRejectsInvalidTrustedProxyCIDR(t *testing.T) {
	if _, err := New(Config{Password: "a sufficiently long password", TrustedProxies: []string{"not-a-cidr"}}); err == nil {
		t.Fatal("invalid trusted proxy CIDR accepted")
	}
}

func requestFrom(remote, forwarded string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	r.RemoteAddr = remote
	if forwarded != "" {
		r.Header.Set("X-Forwarded-For", forwarded)
	}
	return r
}
