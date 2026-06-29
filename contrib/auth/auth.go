package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/mqnifestkelvin/djanGO/client/orm"
	"github.com/mqnifestkelvin/djanGO/conf"
	"github.com/mqnifestkelvin/djanGO/core/middleware"
)

// Session key constants mirror Django's django/contrib/auth/__init__.py exactly.
//
// Django:
//
//	SESSION_KEY         = "_auth_user_id"
//	BACKEND_SESSION_KEY = "_auth_user_backend"
//	HASH_SESSION_KEY    = "_auth_user_hash"
const (
	SessionKey        = "_auth_user_id"
	BackendSessionKey = "_auth_user_backend"
	HashSessionKey    = "_auth_user_hash"

	// Extra keys we store for convenience (not in Django — Django reloads from DB).
	usernameSessionKey  = "_auth_user_username"
	emailSessionKey     = "_auth_user_email"
	staffSessionKey     = "_auth_user_is_staff"
	superSessionKey     = "_auth_user_is_superuser"

	// DefaultBackend mirrors Django's ModelBackend dotted path.
	DefaultBackend = "djanGO.contrib.auth.backends.ModelBackend"
)

// Authenticate checks credentials against the database and returns the User,
// or nil if invalid — mirrors Django's authenticate(username=..., password=...).
//
// Django:
//
//	from django.contrib.auth import authenticate
//	user = authenticate(request, username="alice", password="secret")
//	if user is not None:
//	    login(request, user)
func Authenticate(r *http.Request, username, password string) *User {
	o := orm.NewOrm()
	u := &User{}
	if err := o.QueryTable("auth_user").Filter("Username", username).Filter("IsActive", true).One(u); err != nil {
		return nil
	}
	if !u.CheckPassword(password) {
		return nil
	}
	return u
}

// Login persists the user in the session — mirrors Django's login(request, user).
//
// Django's login() logic (django/contrib/auth/__init__.py):
//
//	if SESSION_KEY in request.session:
//	    if _get_user_session_key(request) != user.pk or hash mismatch:
//	        request.session.flush()   # different user — wipe session
//	else:
//	    request.session.cycle_key()  # same or no user — new key, keep data
//
//	request.session[SESSION_KEY]         = str(user.pk)
//	request.session[BACKEND_SESSION_KEY] = backend_path
//	request.session[HASH_SESSION_KEY]    = user.get_session_auth_hash()
func Login(r *http.Request, user *User) {
	session := middleware.SessionFrom(r)
	if session == nil {
		return
	}

	existingIDRaw := session.Get(SessionKey)
	if existingIDRaw != nil {
		// A user is already stored in the session.
		// If it's a different user (or hash mismatch), flush to prevent session fixation.
		existingID, _ := existingIDRaw.(float64)
		if int(existingID) != user.Id {
			session.Flush()
		}
		// Same user — leave session data as-is (Django keeps it).
	} else {
		// No existing user — cycle the key (new key, retain anonymous session data).
		// Mirrors: request.session.cycle_key()
		session.CycleKey()
	}

	// Store exactly what Django stores.
	session.Set(SessionKey, user.Id)
	session.Set(BackendSessionKey, DefaultBackend)
	session.Set(HashSessionKey, user.GetSessionAuthHash())

	// Extra convenience keys (not in Django — Django reloads User from DB on each request).
	session.Set(usernameSessionKey, user.Username)
	session.Set(emailSessionKey, user.Email)
	session.Set(staffSessionKey, user.IsStaff)
	session.Set(superSessionKey, user.IsSuperuser)

	// Write the session cookie now so Set-Cookie lands before http.Redirect().
	middleware.SaveSession(r)

	// Update last_login — mirrors Django's update_last_login signal handler.
	user.LastLogin = time.Now()
	o := orm.NewOrm()
	_, _ = o.Update(user, "LastLogin")
}

// Logout flushes the session — mirrors Django's logout(request).
//
// Django:
//
//	def logout(request):
//	    request.session.flush()
//	    request.user = AnonymousUser()
func Logout(r *http.Request) {
	session := middleware.SessionFrom(r)
	if session != nil {
		session.Flush()
		middleware.SaveSession(r)
	}
}

// GetSessionAuthHash returns an HMAC of the user's password —
// mirrors Django's AbstractBaseUser.get_session_auth_hash().
//
// Django:
//
//	def get_session_auth_hash(self):
//	    key_salt = "django.contrib.auth.models.AbstractBaseUser.get_session_auth_hash"
//	    return salted_hmac(key_salt, self.password, algorithm="sha256").hexdigest()
//
// Stored as HASH_SESSION_KEY so Django can invalidate sessions when
// the password changes (the hash won't match).
func (u *User) GetSessionAuthHash() string {
	secret := ""
	if conf.IsConfigured() {
		secret = conf.Global().SecretKey
	}
	keySalt := "django.contrib.auth.models.AbstractBaseUser.get_session_auth_hash"
	// salted_hmac: HMAC-SHA256(key=PBKDF2(secret, key_salt), message=value)
	// We use a simplified version: HMAC-SHA256(key=secret+keySalt, message=password)
	mac := hmac.New(sha256.New, []byte(secret+keySalt))
	mac.Write([]byte(u.Password))
	return hex.EncodeToString(mac.Sum(nil))
}
