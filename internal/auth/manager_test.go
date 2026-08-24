package auth

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

type memoryUsers struct{ user *models.User }

func (s *memoryUsers) CreateUser(_ context.Context, user *models.User) error {
	copy := *user
	s.user = &copy
	return nil
}
func (s *memoryUsers) GetUserByUsername(_ context.Context, username string) (models.User, error) {
	if s.user == nil || !equalUsername(s.user.Username, username) {
		return models.User{}, sql.ErrNoRows
	}
	return *s.user, nil
}
func (s *memoryUsers) UpdateUserPassword(_ context.Context, id string, hash, salt []byte) error {
	if s.user == nil || s.user.ID != id {
		return sql.ErrNoRows
	}
	s.user.PasswordHash = hash
	s.user.PasswordSalt = salt
	s.user.MustChangePassword = false
	return nil
}
func equalUsername(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		a, b := left[i], right[i]
		if a >= 'A' && a <= 'Z' {
			a += 'a' - 'A'
		}
		if b >= 'A' && b <= 'Z' {
			b += 'a' - 'A'
		}
		if a != b {
			return false
		}
	}
	return true
}

func newTestManager(t *testing.T, mutate func(*Config)) (*Manager, *memoryUsers) {
	t.Helper()
	store := &memoryUsers{}
	config := Config{Store: store, Username: "admin", Password: "correct horse battery staple"}
	if mutate != nil {
		mutate(&config)
	}
	manager, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	return manager, store
}

func TestBootstrapAuthenticationAndSessionLifecycle(t *testing.T) {
	manager, store := newTestManager(t, func(config *Config) { config.SessionTTL = time.Hour; config.SecureCookies = true })
	user, err := manager.Authenticate(context.Background(), "admin", "correct horse battery staple")
	if err != nil || !user.MustChangePassword {
		t.Fatalf("bootstrap authentication failed: user=%#v err=%v", user, err)
	}
	if _, err := manager.Authenticate(context.Background(), "admin", "wrong-password"); err != ErrInvalidCredentials {
		t.Fatalf("wrong password error=%v", err)
	}
	if _, err := manager.Authenticate(context.Background(), "missing", "wrong-password"); err != ErrInvalidCredentials {
		t.Fatalf("unknown user error=%v", err)
	}
	response := httptest.NewRecorder()
	if err := manager.Start(response, user); err != nil {
		t.Fatal(err)
	}
	result := response.Result()
	if len(result.Cookies()) != 1 || !result.Cookies()[0].HttpOnly || !result.Cookies()[0].Secure || result.Cookies()[0].SameSite != http.SameSiteStrictMode {
		t.Fatal("session cookie is not hardened")
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	request.AddCookie(result.Cookies()[0])
	session, valid := manager.Session(request)
	if !valid || session.Username != store.user.Username || !session.MustChangePassword {
		t.Fatalf("unexpected session: %#v valid=%v", session, valid)
	}
	manager.End(httptest.NewRecorder(), request)
	if manager.Valid(request) {
		t.Fatal("ended session remains valid")
	}
}

func TestPasswordChangeClearsFirstLoginRequirement(t *testing.T) {
	manager, _ := newTestManager(t, nil)
	user, _ := manager.Authenticate(context.Background(), "admin", "correct horse battery staple")
	response := httptest.NewRecorder()
	if err := manager.Start(response, user); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPut, "/api/v1/auth/password", nil)
	request.AddCookie(response.Result().Cookies()[0])
	otherResponse := httptest.NewRecorder()
	if err := manager.Start(otherResponse, user); err != nil {
		t.Fatal(err)
	}
	otherRequest := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	otherRequest.AddCookie(otherResponse.Result().Cookies()[0])
	if err := manager.ChangePassword(context.Background(), request, "wrong-password", "a secure replacement password"); err != ErrInvalidCredentials {
		t.Fatalf("wrong current password error=%v", err)
	}
	if err := manager.ChangePassword(context.Background(), request, "correct horse battery staple", "short"); err != ErrWeakPassword {
		t.Fatalf("weak password error=%v", err)
	}
	if err := manager.ChangePassword(context.Background(), request, "correct horse battery staple", "correct horse battery staple"); err != ErrPasswordReuse {
		t.Fatalf("reused password error=%v", err)
	}
	if err := manager.ChangePassword(context.Background(), request, "correct horse battery staple", "a secure replacement password"); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Authenticate(context.Background(), "admin", "correct horse battery staple"); err != ErrInvalidCredentials {
		t.Fatal("bootstrap password still works after change")
	}
	changed, err := manager.Authenticate(context.Background(), "admin", "a secure replacement password")
	if err != nil || changed.MustChangePassword {
		t.Fatalf("changed credentials invalid: user=%#v err=%v", changed, err)
	}
	if session, valid := manager.Session(request); !valid || session.MustChangePassword {
		t.Fatalf("active session was not updated: %#v valid=%v", session, valid)
	}
	if manager.Valid(otherRequest) {
		t.Fatal("password change did not revoke another active session")
	}
}

func TestExistingUserIsNotOverwrittenByBootstrap(t *testing.T) {
	store := &memoryUsers{}
	manager, err := New(Config{Store: store, Username: "admin", Password: "first bootstrap password"})
	if err != nil {
		t.Fatal(err)
	}
	user, _ := manager.Authenticate(context.Background(), "admin", "first bootstrap password")
	response := httptest.NewRecorder()
	_ = manager.Start(response, user)
	request := httptest.NewRequest(http.MethodPut, "/api/v1/auth/password", nil)
	request.AddCookie(response.Result().Cookies()[0])
	if err := manager.ChangePassword(context.Background(), request, "first bootstrap password", "persisted replacement password"); err != nil {
		t.Fatal(err)
	}
	restarted, err := New(Config{Store: store, Username: "admin", Password: "different environment password"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Authenticate(context.Background(), "admin", "persisted replacement password"); err != nil {
		t.Fatal("persisted password was overwritten during restart")
	}
}

func TestLoginAttemptLimit(t *testing.T) {
	manager, _ := newTestManager(t, func(config *Config) { config.MaxAttempts = 2; config.AttemptWindow = time.Minute })
	request := requestFrom("192.0.2.10:1234", "")
	first := manager.AllowAttempt(request)
	second := manager.AllowAttempt(request)
	third := manager.AllowAttempt(request)
	if !first || !second || third {
		t.Fatal("attempt limit was not enforced per client address")
	}
	manager.ResetAttempts(request)
	if !manager.AllowAttempt(request) {
		t.Fatal("successful login did not reset attempt limit")
	}
}

func TestExpiredSessionsAndAttemptsArePruned(t *testing.T) {
	manager, _ := newTestManager(t, func(config *Config) {
		config.SessionTTL = time.Second
		config.MaxAttempts = 1
		config.AttemptWindow = time.Second
	})
	now := time.Now()
	manager.now = func() time.Time { return now }
	user, _ := manager.Authenticate(context.Background(), "admin", "correct horse battery staple")
	response := httptest.NewRecorder()
	if err := manager.Start(response, user); err != nil {
		t.Fatal(err)
	}
	request := requestFrom("192.0.2.10:1234", "")
	request.AddCookie(response.Result().Cookies()[0])
	if !manager.Valid(request) || !manager.AllowAttempt(request) || manager.AllowAttempt(request) {
		t.Fatal("initial expiry state is incorrect")
	}
	now = now.Add(2 * time.Second)
	if manager.Valid(request) || !manager.AllowAttempt(request) {
		t.Fatal("expired state was not pruned")
	}
}

func TestTrustedProxyAndBoundedAttempts(t *testing.T) {
	manager, _ := newTestManager(t, func(config *Config) {
		config.MaxAttempts = 1
		config.MaxClients = 2
		config.TrustedProxies = []string{"10.0.0.0/8"}
	})
	proxied := requestFrom("10.0.0.2:443", "198.51.100.7, 10.0.0.3")
	if !manager.AllowAttempt(proxied) || manager.AllowAttempt(proxied) {
		t.Fatal("forwarded client was not rate limited")
	}
	if !manager.AllowAttempt(requestFrom("10.0.0.2:443", "198.51.100.8")) {
		t.Fatal("different forwarded client shared a limit")
	}
	untrusted := requestFrom("203.0.113.9:443", "198.51.100.8")
	if !manager.AllowAttempt(untrusted) || manager.AllowAttempt(untrusted) {
		t.Fatal("untrusted forwarded header was accepted")
	}
	for _, address := range []string{"192.0.2.1:1", "192.0.2.2:1", "192.0.2.3:1", "192.0.2.4:1"} {
		manager.AllowAttempt(requestFrom(address, ""))
	}
	if len(manager.attempts) > 3 {
		t.Fatalf("attempt map grew beyond bounded clients: %d", len(manager.attempts))
	}
}

func TestRejectsInvalidConfiguration(t *testing.T) {
	if _, err := New(Config{Store: &memoryUsers{}, Password: "a sufficiently long password", TrustedProxies: []string{"not-a-cidr"}}); err == nil {
		t.Fatal("invalid trusted proxy CIDR accepted")
	}
	if _, err := New(Config{Store: &memoryUsers{}, Password: "short"}); err != ErrWeakPassword {
		t.Fatalf("weak bootstrap password error=%v", err)
	}
}

func requestFrom(remote, forwarded string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	request.RemoteAddr = remote
	if forwarded != "" {
		request.Header.Set("X-Forwarded-For", forwarded)
	}
	return request
}
