package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net"
	"net/http"
	"sync"
	"time"
)

const CookieName = "pgsentinel_session"

type Config struct {
	Password      string
	SecureCookies bool
	SessionTTL    time.Duration
	MaxAttempts   int
	AttemptWindow time.Duration
}

type attemptWindow struct {
	started time.Time
	count   int
}

type Manager struct {
	passwordHash [sha256.Size]byte
	secure       bool
	ttl          time.Duration
	maxAttempts  int
	window       time.Duration

	mu       sync.Mutex
	sessions map[[sha256.Size]byte]time.Time
	attempts map[string]attemptWindow
	now      func() time.Time
}

func New(cfg Config) *Manager {
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = 12 * time.Hour
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 5
	}
	if cfg.AttemptWindow <= 0 {
		cfg.AttemptWindow = 5 * time.Minute
	}
	return &Manager{
		passwordHash: sha256.Sum256([]byte(cfg.Password)), secure: cfg.SecureCookies,
		ttl: cfg.SessionTTL, maxAttempts: cfg.MaxAttempts, window: cfg.AttemptWindow,
		sessions: map[[sha256.Size]byte]time.Time{}, attempts: map[string]attemptWindow{}, now: time.Now,
	}
}

func (m *Manager) CheckPassword(password string) bool {
	candidate := sha256.Sum256([]byte(password))
	return subtle.ConstantTimeCompare(candidate[:], m.passwordHash[:]) == 1
}

func (m *Manager) AllowAttempt(remoteAddr string) bool {
	key := clientAddress(remoteAddr)
	now := m.now()
	m.mu.Lock()
	defer m.mu.Unlock()
	entry := m.attempts[key]
	if entry.started.IsZero() || now.Sub(entry.started) >= m.window {
		entry = attemptWindow{started: now}
	}
	if entry.count >= m.maxAttempts {
		return false
	}
	entry.count++
	m.attempts[key] = entry
	return true
}

func (m *Manager) ResetAttempts(remoteAddr string) {
	m.mu.Lock()
	delete(m.attempts, clientAddress(remoteAddr))
	m.mu.Unlock()
}

func (m *Manager) Start(w http.ResponseWriter) error {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	expires := m.now().Add(m.ttl)
	m.mu.Lock()
	m.pruneLocked()
	m.sessions[hash] = expires
	m.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: token, Path: "/", Expires: expires, MaxAge: int(m.ttl.Seconds()), HttpOnly: true, Secure: m.secure, SameSite: http.SameSiteStrictMode})
	return nil
}

func (m *Manager) Valid(r *http.Request) bool {
	cookie, err := r.Cookie(CookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	hash := sha256.Sum256([]byte(cookie.Value))
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked()
	expires, ok := m.sessions[hash]
	return ok && expires.After(m.now())
}

func (m *Manager) End(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(CookieName); err == nil {
		hash := sha256.Sum256([]byte(cookie.Value))
		m.mu.Lock()
		delete(m.sessions, hash)
		m.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: m.secure, SameSite: http.SameSiteStrictMode})
}

func (m *Manager) pruneLocked() {
	now := m.now()
	for token, expires := range m.sessions {
		if !expires.After(now) {
			delete(m.sessions, token)
		}
	}
}

func clientAddress(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}
	return remoteAddr
}
