// Package sessions mirrors Django's django.contrib.sessions.
//
// Django reference: django/contrib/sessions/models.py,
//                   django/contrib/sessions/backends/db.py
//
// Django stores sessions in the database (django_session table) by default.
// Each row: session_key (40-char random string), session_data (base64+pickle),
// expire_date.
//
// djanGO: same table structure, JSON instead of pickle.
// The session key is stored in the browser cookie ("sessionid").
// Session data lives in the DB row.
//
// Table: django_session
package sessions

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/mqnifestkelvin/djanGO/client/orm"
	"github.com/mqnifestkelvin/djanGO/core/models"
)

// Session mirrors Django's django.contrib.sessions.models.Session.
//
// Django:
//
//	class Session(AbstractBaseSession):
//	    class Meta:
//	        db_table = "django_session"
//
//	session_key   VARCHAR(40) primary key
//	session_data  TextField   (base64-encoded pickle — we use JSON)
//	expire_date   DateTimeField
type Session struct {
	models.Model
	SessionKey  string    `orm:"pk;size(40)"`
	SessionData string    `orm:"type(text)"`
	ExpireDate  time.Time `orm:"type(datetime);index"`
}

func (s *Session) TableName() string { return "django_session" }

// Objects is the default manager.
var Objects = models.NewManager(func() Session { return Session{} })

// validKeyChars mirrors Django's VALID_KEY_CHARS = string.ascii_lowercase + string.digits
// django/contrib/sessions/backends/base.py: VALID_KEY_CHARS = string.ascii_lowercase + string.digits
const validKeyChars = "abcdefghijklmnopqrstuvwxyz0123456789"

// newSessionKey generates a random 32-character session key using [a-z0-9] —
// mirrors Django's SessionBase._get_new_session_key().
//
// Django:
//
//	VALID_KEY_CHARS = string.ascii_lowercase + string.digits  # "abcdefghijklmnopqrstuvwxyz0123456789"
//	session_key = get_random_string(32, VALID_KEY_CHARS)
func newSessionKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	key := make([]byte, 32)
	for i, v := range b {
		key[i] = validKeyChars[int(v)%len(validKeyChars)]
	}
	return string(key), nil
}

// Load retrieves session data from DB by key.
// Returns nil map if not found or expired.
func Load(sessionKey string) (map[string]interface{}, error) {
	o := orm.NewOrm()
	s := &Session{}
	err := o.QueryTable("django_session").
		Filter("SessionKey", sessionKey).
		Filter("ExpireDate__gt", time.Now()).
		One(s)
	if err != nil {
		return nil, nil // expired or not found — return empty session
	}

	raw, err := base64.StdEncoding.DecodeString(s.SessionData)
	if err != nil {
		return nil, nil
	}
	data := map[string]interface{}{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, nil
	}
	return data, nil
}

// Save writes session data to DB, creating the row if needed —
// mirrors Django's SessionStore.save().
func Save(sessionKey string, data map[string]interface{}, expiry time.Duration) (string, error) {
	o := orm.NewOrm()

	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	encoded := base64.StdEncoding.EncodeToString(b)

	if sessionKey == "" {
		sessionKey, err = newSessionKey()
		if err != nil {
			return "", err
		}
	}

	expireDate := time.Now().Add(expiry)
	s := &Session{
		SessionKey:  sessionKey,
		SessionData: encoded,
		ExpireDate:  expireDate,
	}

	existing := &Session{SessionKey: sessionKey}
	if o.Read(existing) == nil {
		s.SessionData = encoded
		s.ExpireDate = expireDate
		_, err = o.Update(s, "SessionData", "ExpireDate")
	} else {
		_, err = o.Insert(s)
	}
	return sessionKey, err
}

// Delete removes a session from DB — mirrors Django's SessionStore.delete().
//
// Django:
//
//	def delete(self, session_key=None):
//	    ...
//	    Session.objects.filter(session_key=self.session_key).delete()
func Delete(sessionKey string) {
	if sessionKey == "" {
		return
	}
	o := orm.NewOrm()
	_, _ = o.Delete(&Session{SessionKey: sessionKey})
}

// CycleKey creates a new session key while keeping the existing session data —
// mirrors Django's SessionStore.cycle_key().
//
// Django:
//
//	def cycle_key(self):
//	    data = self._session
//	    key = self.session_key
//	    self.create()          # new key
//	    self._session_cache = data  # restore data
//	    if key:
//	        self.delete(key)   # delete old row
//
// Called by login() when an anonymous session exists but no user is logged in,
// to prevent session fixation without losing anonymous session data.
func CycleKey(oldKey string, data map[string]interface{}, expiry time.Duration) (string, error) {
	newKey, err := Save("", data, expiry)
	if err != nil {
		return "", err
	}
	Delete(oldKey)
	return newKey, nil
}

func init() {
	orm.RegisterModel(&Session{})
}
