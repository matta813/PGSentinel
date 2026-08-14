package notifications

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebhook(t *testing.T) {
	called := false
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing content type")
		}
		w.WriteHeader(204)
	}))
	defer s.Close()
	p, err := NewWebhook(s.URL, nil, NewTargetPolicy(true, nil))
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Send(context.Background(), Message{Title: "test"}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("not called")
	}
}

func TestBlocksPrivateTargetsByDefault(t *testing.T) {
	if _, err := NewWebhook("http://127.0.0.1:8080/hook", nil); err == nil {
		t.Fatal("expected loopback target rejection")
	}
	if _, err := NewWebhook("http://169.254.169.254/latest/meta-data", nil); err == nil {
		t.Fatal("expected link-local metadata target rejection")
	}
}

func TestExplicitHostAllowlistPermitsPrivateTarget(t *testing.T) {
	if _, err := NewWebhook("http://127.0.0.1:8080/hook", nil, NewTargetPolicy(false, []string{"127.0.0.1"})); err != nil {
		t.Fatal(err)
	}
}

func TestRejectsCredentialsInTargetURL(t *testing.T) {
	if _, err := NewWebhook("https://user:secret@example.com/hook", nil); err == nil {
		t.Fatal("expected URL credentials rejection")
	}
}
func TestRejectsInvalidURL(t *testing.T) {
	if _, err := NewWebhook("file:///etc/passwd", nil); err == nil {
		t.Fatal("expected rejection")
	}
}
