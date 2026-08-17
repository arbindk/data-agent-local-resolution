// COPY THIS FILE TO: internal/api/auth_handlers.go  (REPLACES the existing file)
//
// Register in server.go (inside NewServer / Routes). The first three already
// exist; ADD the two new session routes:
//
//     mux.HandleFunc("/v1/auth/login", s.authLogin)
//     mux.HandleFunc("/v1/auth/register", s.authRegister)
//     mux.HandleFunc("/v1/auth/users", s.authUsers)
//     mux.HandleFunc("/v1/auth/logout", s.authLogout)     // NEW
//     mux.HandleFunc("/v1/auth/session", s.authSession)   // NEW
//
// And wrap the mux so every request refreshes a live session's sliding expiry:
//
//     return s.WithSessionTouch(mux)   // instead of `return mux`
//
// ----------------------------------------------------------------------------
// Production session model
// ----------------------------------------------------------------------------
//   * Login/register issue a server-side opaque bearer token. The token is the
//     ONLY proof of identity; the client can no longer self-assert a role.
//   * Sessions persist to data/auth/sessions.json so they survive agent restart
//     (required for "remember me").
//   * Sliding idle timeout (default 30 min) + an absolute max lifetime. Every
//     authenticated request refreshes the idle window (see WithSessionTouch).
//       - normal      : 30 min idle, 12 h absolute
//       - remember me  : 14 d  idle, 30 d absolute
//   * Passwords use PBKDF2-HMAC-SHA256 (Go stdlib, no new deps). Legacy salted
//     SHA-256 users still verify and are transparently upgraded on next login.
//   * Signup policy: the FIRST account becomes platform_admin (bootstrap); every
//     later self-signup is forced to data_analyst — elevated roles need an admin.
//
// authorization.go derives the trusted actor/role/tenant from the session when a
// valid bearer token is present, and only falls back to X-Everest-* headers for
// unauthenticated callers (agent runtime poll / dev). It still decides WHAT a
// caller may do; this file proves WHO they are.
package api

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// recognized roles must match authorization.go
var validRoles = map[string]bool{
	"platform_admin": true, "security_officer": true, "compliance_officer": true,
	"data_steward": true, "data_analyst": true, "auditor": true, "agent_runtime": true,
}

// roles a user may NOT grant themselves at self-signup (need an admin).
var elevatedRoles = map[string]bool{
	"platform_admin": true, "security_officer": true, "compliance_officer": true,
	"data_steward": true, "agent_runtime": true,
}

const (
	pbkdf2Iter      = 210000
	pbkdf2KeyLen    = 32
	idleTTLDefault  = 30 * time.Minute
	maxLifeDefault  = 12 * time.Hour
	idleTTLRemember = 14 * 24 * time.Hour
	maxLifeRemember = 30 * 24 * time.Hour
)

type authUser struct {
	Username  string `json:"username"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	Tenant    string `json:"tenant"`
	Salt      string `json:"salt"`
	PassHash  string `json:"pass_hash"`
	Algo      string `json:"algo,omitempty"` // "pbkdf2" (new) or "" (legacy sha256)
	Iter      int    `json:"iter,omitempty"`
	CreatedAt string `json:"created_at"`
}

type authSession struct {
	Token     string    `json:"token"`
	Username  string    `json:"username"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Tenant    string    `json:"tenant"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
	IdleAt    time.Time `json:"idle_at"` // sliding deadline
	MaxAt     time.Time `json:"max_at"`  // absolute deadline
	IdleTTL   int64     `json:"idle_ttl_ns"`
	Remember  bool      `json:"remember"`
}

type authStore struct {
	mu        sync.Mutex
	usersPath string
	sessPath  string
	users     map[string]authUser
	sessions  map[string]authSession
}

var authStoreOnce sync.Once
var authStoreInst *authStore

func (s *Server) auth() *authStore {
	authStoreOnce.Do(func() {
		st := &authStore{
			usersPath: filepath.Join("data", "auth", "users.json"),
			sessPath:  filepath.Join("data", "auth", "sessions.json"),
			users:     map[string]authUser{},
			sessions:  map[string]authSession{},
		}
		if b, err := os.ReadFile(st.usersPath); err == nil {
			var list []authUser
			if json.Unmarshal(b, &list) == nil {
				for _, u := range list {
					st.users[strings.ToLower(u.Username)] = u
				}
			}
		}
		if b, err := os.ReadFile(st.sessPath); err == nil {
			var list []authSession
			if json.Unmarshal(b, &list) == nil {
				now := time.Now().UTC()
				for _, sess := range list {
					if now.Before(sess.IdleAt) && now.Before(sess.MaxAt) {
						st.sessions[sess.Token] = sess
					}
				}
			}
		}
		authStoreInst = st
	})
	return authStoreInst
}

// callers must hold st.mu
func (st *authStore) saveUsers() {
	_ = os.MkdirAll(filepath.Dir(st.usersPath), 0o755)
	list := make([]authUser, 0, len(st.users))
	for _, u := range st.users {
		list = append(list, u)
	}
	if b, err := json.MarshalIndent(list, "", "  "); err == nil {
		_ = os.WriteFile(st.usersPath, b, 0o600)
	}
}

// callers must hold st.mu
func (st *authStore) saveSessions() {
	_ = os.MkdirAll(filepath.Dir(st.sessPath), 0o755)
	list := make([]authSession, 0, len(st.sessions))
	for _, sess := range st.sessions {
		list = append(list, sess)
	}
	if b, err := json.MarshalIndent(list, "", "  "); err == nil {
		_ = os.WriteFile(st.sessPath, b, 0o600)
	}
}

func hashPassPBKDF2(salt, pass string, iter int) string {
	dk, err := pbkdf2.Key(sha256.New, pass, []byte(salt), iter, pbkdf2KeyLen)
	if err != nil {
		// pbkdf2.Key only errors on absurd params; fall back to a fixed iter.
		dk, _ = pbkdf2.Key(sha256.New, pass, []byte(salt), pbkdf2Iter, pbkdf2KeyLen)
	}
	return hex.EncodeToString(dk)
}

// legacy salted SHA-256 (kept only to verify + upgrade old accounts)
func hashPassLegacy(salt, pass string) string {
	sum := sha256.Sum256([]byte(salt + ":" + pass))
	return hex.EncodeToString(sum[:])
}

func newSalt() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func newToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func constEq(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// verifyPassword returns (ok, needsUpgrade).
func verifyPassword(u authUser, pass string) (bool, bool) {
	if u.Algo == "pbkdf2" {
		iter := u.Iter
		if iter == 0 {
			iter = pbkdf2Iter
		}
		return constEq(u.PassHash, hashPassPBKDF2(u.Salt, pass, iter)), false
	}
	// legacy
	if constEq(u.PassHash, hashPassLegacy(u.Salt, pass)) {
		return true, true
	}
	return false, false
}

// ----------------------------------------------------------------------------
// session helpers
// ----------------------------------------------------------------------------

func bearerToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		if strings.HasPrefix(strings.ToLower(h), "bearer ") {
			return strings.TrimSpace(h[7:])
		}
	}
	return strings.TrimSpace(r.Header.Get("X-Everest-Token"))
}

// callers must hold st.mu
func (st *authStore) issueSession(u authUser, remember bool) authSession {
	now := time.Now().UTC()
	idle := idleTTLDefault
	maxLife := maxLifeDefault
	if remember {
		idle = idleTTLRemember
		maxLife = maxLifeRemember
	}
	sess := authSession{
		Token:    newToken(),
		Username: u.Username, Name: u.Name, Role: u.Role, Tenant: u.Tenant,
		CreatedAt: now, LastSeen: now,
		IdleAt: now.Add(idle), MaxAt: now.Add(maxLife),
		IdleTTL: int64(idle), Remember: remember,
	}
	st.sessions[sess.Token] = sess
	st.saveSessions()
	return sess
}

// lookupSession validates and (touch=true) slides the idle window. Returns a
// copy of the live session. Expired/unknown tokens return ok=false.
func (st *authStore) lookupSession(token string, touch bool) (authSession, bool) {
	if token == "" {
		return authSession{}, false
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	sess, ok := st.sessions[token]
	if !ok {
		return authSession{}, false
	}
	now := time.Now().UTC()
	if !now.Before(sess.IdleAt) || !now.Before(sess.MaxAt) {
		delete(st.sessions, token)
		st.saveSessions()
		return authSession{}, false
	}
	if touch {
		sess.LastSeen = now
		newIdle := now.Add(time.Duration(sess.IdleTTL))
		if newIdle.After(sess.MaxAt) {
			newIdle = sess.MaxAt
		}
		sess.IdleAt = newIdle
		st.sessions[token] = sess
		st.saveSessions()
	}
	return sess, true
}

// sessionActor is consumed by authorization.go to derive a trusted identity.
func (s *Server) sessionActor(r *http.Request) (enterpriseActor, bool) {
	sess, ok := s.auth().lookupSession(bearerToken(r), false)
	if !ok {
		return enterpriseActor{}, false
	}
	return enterpriseActor{Actor: sess.Username, Role: sess.Role, Tenant: sess.Tenant}, true
}

// WithSessionTouch refreshes a valid session's sliding window on every request.
// It never blocks a request — it only extends activity-based expiry so an
// actively-used portal does not get logged out mid-session.
func (s *Server) WithSessionTouch(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tok := bearerToken(r); tok != "" {
			_, _ = s.auth().lookupSession(tok, true)
		}
		next.ServeHTTP(w, r)
	})
}

func sessionPayload(sess authSession) map[string]any {
	return map[string]any{
		"status": "ok", "username": sess.Username, "name": sess.Name,
		"role": sess.Role, "tenant": sess.Tenant, "token": sess.Token,
		"verified": true, "remember": sess.Remember,
		"expires_at":      sess.MaxAt.Format(time.RFC3339),
		"idle_expires_at": sess.IdleAt.Format(time.RFC3339),
	}
}

// ----------------------------------------------------------------------------
// handlers
// ----------------------------------------------------------------------------

// POST /v1/auth/register {username,password,name,role,tenant,remember}
func (s *Server) authRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Username, Password, Name, Role, Tenant string
		Remember                               bool
	}
	if err := jsonDecode(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))
	if req.Username == "" || strings.TrimSpace(req.Password) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password are required"})
		return
	}
	if len(req.Password) < 8 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters"})
		return
	}

	st := s.auth()
	st.mu.Lock()
	defer st.mu.Unlock()

	if _, ok := st.users[req.Username]; ok {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "user already exists"})
		return
	}

	// Signup policy: first account bootstraps as platform_admin; later self
	// signups are forced to data_analyst.
	bootstrap := len(st.users) == 0
	note := ""
	role := strings.TrimSpace(req.Role)
	if bootstrap {
		if role == "" || !validRoles[role] {
			role = "platform_admin"
		}
	} else {
		if role != "" && role != "data_analyst" && elevatedRoles[role] {
			note = "Account created as data_analyst. Elevated roles must be granted by an administrator."
		}
		role = "data_analyst"
	}

	salt := newSalt()
	u := authUser{
		Username: req.Username, Name: strings.TrimSpace(req.Name), Role: role,
		Tenant: strings.TrimSpace(req.Tenant), Salt: salt,
		PassHash: hashPassPBKDF2(salt, req.Password, pbkdf2Iter), Algo: "pbkdf2", Iter: pbkdf2Iter,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	st.users[req.Username] = u
	st.saveUsers()

	// auto-login the new account
	sess := st.issueSession(u, req.Remember)
	out := sessionPayload(sess)
	out["bootstrap"] = bootstrap
	if note != "" {
		out["note"] = note
	}
	writeJSON(w, http.StatusOK, out)
}

// POST /v1/auth/login {username,password,remember}
func (s *Server) authLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Username, Password string
		Remember           bool
	}
	if err := jsonDecode(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	st := s.auth()
	st.mu.Lock()
	defer st.mu.Unlock()

	u, ok := st.users[strings.ToLower(strings.TrimSpace(req.Username))]
	valid, needsUpgrade := false, false
	if ok {
		valid, needsUpgrade = verifyPassword(u, req.Password)
	}
	if !ok || !valid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid username or password"})
		return
	}
	if needsUpgrade {
		u.Salt = newSalt()
		u.PassHash = hashPassPBKDF2(u.Salt, req.Password, pbkdf2Iter)
		u.Algo = "pbkdf2"
		u.Iter = pbkdf2Iter
		st.users[u.Username] = u
		st.saveUsers()
	}
	sess := st.issueSession(u, req.Remember)
	writeJSON(w, http.StatusOK, sessionPayload(sess))
}

// POST /v1/auth/logout  (Authorization: Bearer <token>) — revoke this session
func (s *Server) authLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	tok := bearerToken(r)
	st := s.auth()
	st.mu.Lock()
	if _, ok := st.sessions[tok]; ok {
		delete(st.sessions, tok)
		st.saveSessions()
	}
	st.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// GET /v1/auth/session  (Authorization: Bearer <token>) — validate + slide
func (s *Server) authSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	sess, ok := s.auth().lookupSession(bearerToken(r), true)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "no active session"})
		return
	}
	writeJSON(w, http.StatusOK, sessionPayload(sess))
}

// GET /v1/auth/users  (platform_admin / security_officer) — Users & Roles screen
func (s *Server) authUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	actor := s.enterpriseActor(r) // session-derived role when a token is present
	if actor.Role != "platform_admin" && actor.Role != "security_officer" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "platform_admin or security_officer required"})
		return
	}
	st := s.auth()
	st.mu.Lock()
	defer st.mu.Unlock()
	items := make([]map[string]any, 0, len(st.users))
	for _, u := range st.users {
		items = append(items, map[string]any{
			"username": u.Username, "name": u.Name, "role": u.Role,
			"tenant": u.Tenant, "created_at": u.CreatedAt, "verified": true,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
}
