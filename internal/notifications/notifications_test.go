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
	p, err := NewWebhook(s.URL, nil)
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
func TestRejectsInvalidURL(t *testing.T) {
	if _, err := NewWebhook("file:///etc/passwd", nil); err == nil {
		t.Fatal("expected rejection")
	}
}
