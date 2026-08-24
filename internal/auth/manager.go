package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/matta813/pgsentinel/internal/models"
	"golang.org/x/crypto/argon2"
)

const CookieName = "pgsentinel_session"

const MinimumPasswordLength = 12

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrWeakPassword       = errors.New("password must contain at least 12 characters")
	ErrPasswordReuse      = errors.New("new password must differ from the current password")
)

type UserStore interface {
	CreateUser(context.Context, *models.User) error
	GetUserByUsername(context.Context, string) (models.User, error)
	UpdateUserPassword(context.Context, string, []byte, []byte) error
}

type Config struct {
	Store          UserStore
	Username       string
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
	store          UserStore
	dummyHash      []byte
	dummySalt      []byte
	secure         bool
	ttl            time.Duration
	maxAttempts    int
	window         time.Duration
	maxClients     int
	trustedProxies []netip.Prefix
	passwordSlots  chan struct{}

	mu       sync.Mutex
	sessions map[[sha256.Size]byte]Session
	attempts map[string]attemptWindow
	now      func() time.Time
}

type Session struct {
	UserID             string    `json:"-"`
	Username           string    `json:"username"`
	MustChangePassword bool      `json:"mustChangePassword"`
	ExpiresAt          time.Time `json:"-"`
}

func New(cfg Config) (*Manager, error) {
	if cfg.Store == nil {
		return nil, errors.New("user store is required")
	}
	if strings.TrimSpace(cfg.Username) == "" {
		cfg.Username = "admin"
	}
	if len(cfg.Password) < MinimumPasswordLength {
		return nil, ErrWeakPassword
	}
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
	dummySalt, err := randomSalt()
	if err != nil {
		return nil, fmt.Errorf("generate password salt: %w", err)
	}
	m := &Manager{
		store: cfg.Store, dummyHash: derivePassword("timing-resistant-placeholder", dummySalt), dummySalt: dummySalt, secure: cfg.SecureCookies,
		ttl: cfg.SessionTTL, maxAttempts: cfg.MaxAttempts, window: cfg.AttemptWindow, maxClients: cfg.MaxClients, trustedProxies: trusted,
		passwordSlots: make(chan struct{}, 2), sessions: map[[sha256.Size]byte]Session{}, attempts: map[string]attemptWindow{}, now: time.Now,
	}
	if err := m.ensureBootstrapUser(context.Background(), strings.TrimSpace(cfg.Username), cfg.Password); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) Authenticate(ctx context.Context, username, password string) (models.User, error) {
	m.passwordSlots <- struct{}{}
	defer func() { <-m.passwordSlots }()
	user, err := m.store.GetUserByUsername(ctx, strings.TrimSpace(username))
	hash, salt := m.dummyHash, m.dummySalt
	if err == nil {
		hash, salt = user.PasswordHash, user.PasswordSalt
	}
	candidate := derivePassword(password, salt)
	if err != nil || subtle.ConstantTimeCompare(candidate, hash) != 1 {
		return models.User{}, ErrInvalidCredentials
	}
	return user, nil
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

func (m *Manager) Start(w http.ResponseWriter, user models.User) error {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	expires := m.now().Add(m.ttl)
	m.mu.Lock()
	m.pruneLocked()
	m.sessions[hash] = Session{UserID: user.ID, Username: user.Username, MustChangePassword: user.MustChangePassword, ExpiresAt: expires}
	m.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: token, Path: "/", Expires: expires, MaxAge: int(m.ttl.Seconds()), HttpOnly: true, Secure: m.secure, SameSite: http.SameSiteStrictMode})
	return nil
}

func (m *Manager) Valid(r *http.Request) bool {
	_, ok := m.Session(r)
	return ok
}

func (m *Manager) Session(r *http.Request) (Session, bool) {
	cookie, err := r.Cookie(CookieName)
	if err != nil || cookie.Value == "" {
		return Session{}, false
	}
	hash := sha256.Sum256([]byte(cookie.Value))
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked()
	session, ok := m.sessions[hash]
	return session, ok && session.ExpiresAt.After(m.now())
}

func (m *Manager) ChangePassword(ctx context.Context, r *http.Request, currentPassword, newPassword string) error {
	if len(newPassword) < MinimumPasswordLength {
		return ErrWeakPassword
	}
	session, ok := m.Session(r)
	if !ok {
		return ErrInvalidCredentials
	}
	cookie, err := r.Cookie(CookieName)
	if err != nil || cookie.Value == "" {
		return ErrInvalidCredentials
	}
	currentToken := sha256.Sum256([]byte(cookie.Value))
	user, err := m.Authenticate(ctx, session.Username, currentPassword)
	if err != nil {
		return ErrInvalidCredentials
	}
	if subtle.ConstantTimeCompare(derivePassword(newPassword, user.PasswordSalt), user.PasswordHash) == 1 {
		return ErrPasswordReuse
	}
	salt, err := randomSalt()
	if err != nil {
		return fmt.Errorf("generate password salt: %w", err)
	}
	if err := m.store.UpdateUserPassword(ctx, user.ID, derivePassword(newPassword, salt), salt); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	m.mu.Lock()
	for token, active := range m.sessions {
		if active.UserID != user.ID {
			continue
		}
		if token != currentToken {
			delete(m.sessions, token)
			continue
		}
		active.MustChangePassword = false
		m.sessions[token] = active
	}
	m.mu.Unlock()
	return nil
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
	for token, session := range m.sessions {
		if !session.ExpiresAt.After(now) {
			delete(m.sessions, token)
		}
	}
}

func (m *Manager) ensureBootstrapUser(ctx context.Context, username, password string) error {
	if _, err := m.store.GetUserByUsername(ctx, username); err == nil {
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("load bootstrap user: %w", err)
	}
	salt, err := randomSalt()
	if err != nil {
		return fmt.Errorf("generate bootstrap password salt: %w", err)
	}
	user := models.User{ID: uuid.NewString(), Username: username, PasswordHash: derivePassword(password, salt), PasswordSalt: salt, MustChangePassword: true}
	if err := m.store.CreateUser(ctx, &user); err != nil {
		return fmt.Errorf("create bootstrap user: %w", err)
	}
	return nil
}

func randomSalt() ([]byte, error) {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	return salt, err
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
