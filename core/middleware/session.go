package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/mqnifestkelvin/djanGO/contrib/sessions"
)

type contextKey int

const (
	sessionKey contextKey = iota
	userKey
)

// Session mirrors Django's request.session — a dict-like object backed by the
// database (django_session table) with just the session key in the browser cookie.
//
// Django:
//
//	request.session["key"] = value
//	value = request.session.get("key", None)
//	del request.session["key"]
//	request.session.flush()
//
// Django:
//   - Session key stored in "sessionid" cookie (40-char random string)
//   - Session data stored in django_session.session_data (base64+pickle; we use JSON)
//   - SessionMiddleware.process_response saves if modified
type Session struct {
	data       map[string]interface{}
	sessionKey string // the 40-char key stored in the cookie
	Modified   bool
	w          http.ResponseWriter
}

// SESSION_COOKIE_AGE mirrors Django's SESSION_COOKIE_AGE = 1209600 (2 weeks in seconds).
const sessionCookieAge = 14 * 24 * time.Hour

func newSession(r *http.Request, w http.ResponseWriter) *Session {
	s := &Session{data: make(map[string]interface{}), w: w}

	// Read the session key from the cookie — mirrors Django's
	// SessionMiddleware.process_request reading request.COOKIES[SESSION_COOKIE_NAME].
	cookie, err := r.Cookie("sessionid")
	if err == nil && cookie.Value != "" {
		s.sessionKey = cookie.Value
		// Load session data from DB by key.
		data, _ := sessions.Load(s.sessionKey)
		if data != nil {
			s.data = data
		}
	}
	return s
}

// Get mirrors Django's request.session.get(key, default=None).
func (s *Session) Get(key string) interface{} {
	return s.data[key]
}

// Set mirrors Django's request.session[key] = value.
func (s *Session) Set(key string, value interface{}) {
	s.data[key] = value
	s.Modified = true
}

// Delete mirrors Django's del request.session[key].
func (s *Session) Delete(key string) {
	delete(s.data, key)
	s.Modified = true
}

// Flush clears all session data, deletes the DB row, and sets the key to "" —
// mirrors Django's request.session.flush().
//
// Django:
//
//	def flush(self):
//	    self.clear()       # empties _session_cache
//	    self.delete()      # deletes the DB row
//	    self._session_key = None  # key will be regenerated on next save
func (s *Session) Flush() {
	sessions.Delete(s.sessionKey) // delete DB row
	s.sessionKey = ""             // key = None (new key issued on next save)
	s.data = make(map[string]interface{})
	s.Modified = true
}

// CycleKey creates a new session key while keeping the existing session data —
// mirrors Django's request.session.cycle_key().
//
// Django:
//
//	def cycle_key(self):
//	    data = self._session
//	    key = self.session_key
//	    self.create()
//	    self._session_cache = data
//	    if key: self.delete(key)
//
// Called by Login() when no user is currently logged in (anonymous session),
// to prevent session fixation without losing anonymous session data.
func (s *Session) CycleKey() {
	if s.sessionKey == "" {
		s.Modified = true
		return
	}
	newKey, err := sessions.CycleKey(s.sessionKey, s.data, sessionCookieAge)
	if err != nil {
		return
	}
	s.sessionKey = newKey
	s.Modified = true
}

// save persists the session to the DB and writes the key cookie —
// mirrors Django's SessionMiddleware.process_response.
//
// Django:
//
//	if response.status_code != 500 and modified:
//	    session.save()
//	    response.set_cookie(SESSION_COOKIE_NAME, session.session_key, ...)
func (s *Session) save() {
	if !s.Modified {
		return
	}
	key, err := sessions.Save(s.sessionKey, s.data, sessionCookieAge)
	if err != nil {
		return
	}
	s.sessionKey = key
	http.SetCookie(s.w, &http.Cookie{
		Name:     "sessionid",
		Value:    key,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(sessionCookieAge),
	})
}

// WithSession attaches a new Session to the context.
func WithSession(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
	s := newSession(r, w)
	return context.WithValue(ctx, sessionKey, s)
}

// SessionFrom retrieves the session from the request context —
// mirrors accessing request.session in a Django view.
func SessionFrom(r *http.Request) *Session {
	s, _ := r.Context().Value(sessionKey).(*Session)
	if s == nil {
		return &Session{data: make(map[string]interface{})}
	}
	return s
}

// SaveSession writes the session to the DB if modified.
// Called explicitly by auth.Login/Logout so the Set-Cookie header is
// written before http.Redirect() sends the response.
func SaveSession(r *http.Request) {
	if s, ok := r.Context().Value(sessionKey).(*Session); ok {
		s.save()
	}
}
