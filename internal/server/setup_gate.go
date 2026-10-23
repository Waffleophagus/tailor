package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

const (
	setupCookieName  = "tailor_setup"
	setupSessionTTL  = time.Hour
	maxSetupSessions = 16
)

// SetupGate protects the tsnet node's login link on the setup page. Tailor prints a
// one-off link with the token in the URL fragment (never sent to servers, so
// reverse proxies cannot log it); the page exchanges it for a short-lived
// HttpOnly session cookie. Reset rotates the token and drops every session once
// the node is logged in, so leaked tokens or cookies stop working.
type SetupGate struct {
	mu       sync.Mutex
	token    string
	sessions map[string]time.Time
}

// NewSetupGate returns a gate with a fresh random token.
func NewSetupGate() *SetupGate {
	gate := &SetupGate{}
	gate.Reset()
	return gate
}

// Token is the current setup token, for printing in Tailor's own logs only.
func (g *SetupGate) Token() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.token
}

// Reset rotates the token and invalidates every setup session.
func (g *SetupGate) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.token = randomHex()
	g.sessions = map[string]time.Time{}
}

func (g *SetupGate) exchange(token string) (string, bool) {
	if g == nil || token == "" {
		return "", false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if subtle.ConstantTimeCompare([]byte(token), []byte(g.token)) != 1 {
		return "", false
	}
	now := time.Now()
	for id, expires := range g.sessions {
		if now.After(expires) {
			delete(g.sessions, id)
		}
	}
	if len(g.sessions) >= maxSetupSessions {
		return "", false
	}
	id := randomHex()
	g.sessions[id] = now.Add(setupSessionTTL)
	return id, true
}

func (g *SetupGate) validSession(id string) bool {
	if g == nil || id == "" {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	expires, ok := g.sessions[id]
	return ok && time.Now().Before(expires)
}

// handleSetupExchange swaps the setup token (POSTed by the setup page from the URL
// fragment) for a session cookie.
func handleSetupExchange(w http.ResponseWriter, r *http.Request, gate *SetupGate) {
	if r.Method != http.MethodPost || !fromPrivateNetwork(r.RemoteAddr) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&body); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id, ok := gate.exchange(body.Token)
	if !ok {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     setupCookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(setupSessionTTL / time.Second),
	})
	w.WriteHeader(http.StatusNoContent)
}

func randomHex() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic("setup token generation failed: " + err.Error())
	}
	return hex.EncodeToString(buf[:])
}
