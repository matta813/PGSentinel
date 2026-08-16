package notifications

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type staticResolver []net.IPAddr

func (r staticResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) { return r, nil }

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
	for _, target := range []string{
		"http://127.0.0.1:8080/hook",
		"http://169.254.169.254/latest/meta-data",
		"http://[::1]:8080/hook",
		"http://[fc00::1]:8080/hook",
		"http://[fe80::1]:8080/hook",
	} {
		if _, err := NewWebhook(target, nil); err == nil {
			t.Errorf("expected target rejection: %s", target)
		}
	}
}

func TestExplicitHostAllowlistPermitsPrivateTarget(t *testing.T) {
	if _, err := NewWebhook("http://127.0.0.1:8080/hook", nil, NewTargetPolicy(false, []string{" 127.0.0.1. "})); err != nil {
		t.Fatal(err)
	}
}

func TestDNSAndRedirectCannotReachPrivateTargets(t *testing.T) {
	policy := NewTargetPolicy(false, nil)
	policy.resolver = staticResolver{{IP: net.ParseIP("127.0.0.1")}}
	policy.dialer = &net.Dialer{}
	if _, err := policy.dialContext(context.Background(), "tcp", "public.example:443"); err == nil {
		t.Fatal("DNS resolution to loopback was accepted")
	}
	target, _ := url.Parse("http://169.254.169.254/latest/meta-data")
	if err := policy.checkRedirect(&http.Request{URL: target}, nil); err == nil {
		t.Fatal("redirect to metadata address was accepted")
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
