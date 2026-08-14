package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionLifecycle(t *testing.T) {
	m := New(Config{Password: "correct horse battery staple", SessionTTL: time.Hour})
	if !m.CheckPassword("correct horse battery staple") || m.CheckPassword("wrong") {
		t.Fatal("password comparison failed")
	}
	w := httptest.NewRecorder()
	if err := m.Start(w); err != nil {
		t.Fatal(err)
	}
	result := w.Result()
	if len(result.Cookies()) != 1 || !result.Cookies()[0].HttpOnly || result.Cookies()[0].SameSite != http.SameSiteStrictMode {
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
	m := New(Config{Password: "a sufficiently long password", MaxAttempts: 2, AttemptWindow: time.Minute})
	if !m.AllowAttempt("192.0.2.10:1234") || !m.AllowAttempt("192.0.2.10:5678") || m.AllowAttempt("192.0.2.10:9999") {
		t.Fatal("attempt limit was not enforced per client address")
	}
	m.ResetAttempts("192.0.2.10:1234")
	if !m.AllowAttempt("192.0.2.10:1234") {
		t.Fatal("successful login did not reset attempt limit")
	}
}
