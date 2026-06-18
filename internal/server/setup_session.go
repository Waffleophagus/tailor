package server

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

const setupSessionTTL = 5 * time.Minute
const setupCookieName = "tailor_setup"

type SetupSessions struct {
	mu       sync.Mutex
	sessions map[string]bootstrapSession
}

func NewSetupSessions() *SetupSessions {
	return &SetupSessions{sessions: map[string]bootstrapSession{}}
}

func (s *SetupSessions) Create(loginName, nodeName string) (string, time.Time) {
	token := newBootstrapToken()
	expiresAt := time.Now().Add(setupSessionTTL)
	s.mu.Lock()
	s.purgeExpiredLocked(time.Now())
	s.sessions[token] = bootstrapSession{loginName: loginName, nodeName: nodeName, expiresAt: expiresAt}
	s.mu.Unlock()
	return token, expiresAt
}

func (s *SetupSessions) Consume(token, loginName, nodeName string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(time.Now())
	session, ok := s.sessions[token]
	if !ok || time.Now().After(session.expiresAt) || session.loginName != loginName || session.nodeName != nodeName {
		if ok {
			delete(s.sessions, token)
		}
		return false
	}
	delete(s.sessions, token)
	return true
}

func (s *SetupSessions) purgeExpiredLocked(now time.Time) {
	for token, session := range s.sessions {
		if now.After(session.expiresAt) {
			delete(s.sessions, token)
		}
	}
}

func setSetupCookie(w http.ResponseWriter, r *http.Request, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     setupCookieName,
		Value:    token,
		Path:     "/api/cloud/setup-grant",
		HttpOnly: true,
		Secure:   cookieSecure(r),
		SameSite: http.SameSiteStrictMode,
		Expires:  expiresAt,
	})
}

func cookieSecure(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func setupTokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(setupCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}
