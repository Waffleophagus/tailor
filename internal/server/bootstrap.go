package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"
)

const bootstrapSessionTTL = 15 * time.Minute

type bootstrapSession struct {
	loginName string
	nodeName  string
	expiresAt time.Time
}

type BootstrapSessions struct {
	mu       sync.Mutex
	sessions map[string]bootstrapSession
}

func NewBootstrapSessions() *BootstrapSessions {
	return &BootstrapSessions{sessions: map[string]bootstrapSession{}}
}

func (b *BootstrapSessions) Create(loginName, nodeName string) (token string, expiresAt time.Time) {
	token = newBootstrapToken()
	expiresAt = time.Now().Add(bootstrapSessionTTL)
	b.mu.Lock()
	b.purgeExpiredLocked(time.Now())
	b.sessions[token] = bootstrapSession{
		loginName: loginName,
		nodeName:  nodeName,
		expiresAt: expiresAt,
	}
	b.mu.Unlock()
	return token, expiresAt
}

func (b *BootstrapSessions) Valid(token, loginName, nodeName string) (bool, time.Time) {
	token = strings.TrimSpace(token)
	if token == "" {
		return false, time.Time{}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.purgeExpiredLocked(time.Now())
	session, ok := b.sessions[token]
	if !ok {
		return false, time.Time{}
	}
	if time.Now().After(session.expiresAt) {
		delete(b.sessions, token)
		return false, time.Time{}
	}
	if session.loginName != loginName || session.nodeName != nodeName {
		return false, time.Time{}
	}
	return true, session.expiresAt
}

func (b *BootstrapSessions) purgeExpiredLocked(now time.Time) {
	for token, session := range b.sessions {
		if now.After(session.expiresAt) {
			delete(b.sessions, token)
		}
	}
}

const bootstrapCookieName = "tailor_bootstrap"

func setBootstrapCookie(w http.ResponseWriter, r *http.Request, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     bootstrapCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   cookieSecure(r),
		SameSite: http.SameSiteStrictMode,
		Expires:  expiresAt,
	})
}

func bootstrapTokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(bootstrapCookieName)
	if err != nil || cookie == nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func newBootstrapToken() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic("bootstrap token generation failed: " + err.Error())
	}
	return hex.EncodeToString(buf[:])
}
