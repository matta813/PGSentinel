package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
)

const CookieName = "pgsentinel_session"

type Config struct {
	Password       string
	SecureCookies  bool
	SessionTTL     time.Duration
	MaxAttempts    int
	AttemptWindow  time.Duration
	MaxClients     int
	TrustedProxies []string
}

type attemptWindow struct {
	started time.Time
	count   int
}

type Manager struct {
	passwordHash   []byte
	passwordSalt   []byte
	secure         bool
	ttl            time.Duration
	maxAttempts    int
	window         time.Duration
	maxClients     int
	trustedProxies []netip.Prefix
	passwordSlots  chan struct{}

	mu       sync.Mutex
	sessions map[[sha256.Size]byte]time.Time
	attempts map[string]attemptWindow
	now      func() time.Time
}

func New(cfg Config) (*Manager, error) {
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = 12 * time.Hour
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 5
	}
	if cfg.AttemptWindow <= 0 {
		cfg.AttemptWindow = 5 * time.Minute
	}
	if cfg.MaxClients <= 0 {
		cfg.MaxClients = 10_000
	}
	trusted := make([]netip.Prefix, 0, len(cfg.TrustedProxies))
	for _, raw := range cfg.TrustedProxies {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy CIDR %q: %w", raw, err)
		}
		trusted = append(trusted, prefix.Masked())
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate password salt: %w", err)
	}
	return &Manager{
		passwordHash: derivePassword(cfg.Password, salt), passwordSalt: salt, secure: cfg.SecureCookies,
		ttl: cfg.SessionTTL, maxAttempts: cfg.MaxAttempts, window: cfg.AttemptWindow, maxClients: cfg.MaxClients, trustedProxies: trusted,
		passwordSlots: make(chan struct{}, 2), sessions: map[[sha256.Size]byte]time.Time{}, attempts: map[string]attemptWindow{}, now: time.Now,
	}, nil
}

func (m *Manager) CheckPassword(password string) bool {
	m.passwordSlots <- struct{}{}
	defer func() { <-m.passwordSlots }()
	candidate := derivePassword(password, m.passwordSalt)
	return subtle.ConstantTimeCompare(candidate, m.passwordHash) == 1
}

func (m *Manager) AllowAttempt(r *http.Request) bool {
	key := m.clientAddress(r)
	now := m.now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneAttemptsLocked(now)
	if _, exists := m.attempts[key]; !exists && len(m.attempts) >= m.maxClients {
		key = "__overflow__"
	}
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

func (m *Manager) ResetAttempts(r *http.Request) {
	m.mu.Lock()
	delete(m.attempts, m.clientAddress(r))
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

func (m *Manager) pruneAttemptsLocked(now time.Time) {
	for address, entry := range m.attempts {
		if now.Sub(entry.started) >= m.window {
			delete(m.attempts, address)
		}
	}
}

func (m *Manager) clientAddress(r *http.Request) string {
	peer := parseAddress(r.RemoteAddr)
	if !m.isTrustedProxy(peer) {
		return peer.String()
	}
	forwarded := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for i := len(forwarded) - 1; i >= 0; i-- {
		candidate, err := netip.ParseAddr(strings.TrimSpace(forwarded[i]))
		if err != nil {
			continue
		}
		candidate = candidate.Unmap()
		if !m.isTrustedProxy(candidate) {
			return candidate.String()
		}
	}
	return peer.String()
}

func (m *Manager) isTrustedProxy(address netip.Addr) bool {
	for _, prefix := range m.trustedProxies {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func parseAddress(remote string) netip.Addr {
	if host, _, err := net.SplitHostPort(remote); err == nil {
		remote = host
	}
	address, _ := netip.ParseAddr(remote)
	return address.Unmap()
}

func derivePassword(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, 3, 256*1024, 4, 32)
}
