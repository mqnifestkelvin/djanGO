package middleware

import (
	"context"
	"net/http"
)

// RequestUser mirrors Django's request.user — attached by AuthenticationMiddleware.
//
// Django:
//
//	request.user          # AnonymousUser or User instance
//	request.user.is_authenticated  # True if logged in
//	request.user.username
type RequestUser struct {
	ID              int
	Username        string
	Email           string
	IsAuthenticated bool
	IsStaff         bool
	IsSuperuser     bool
}

// AnonymousUser mirrors Django's AnonymousUser — returned when no session user.
var AnonymousUser = &RequestUser{IsAuthenticated: false}

// WithUser attaches the current user (from session) to the request context —
// mirrors Django's AuthenticationMiddleware.process_request.
func WithUser(ctx context.Context, r *http.Request) context.Context {
	session := SessionFrom(r)

	userID, _ := session.Get("_auth_user_id").(float64) // JSON numbers decode as float64
	if userID == 0 {
		return context.WithValue(ctx, userKey, AnonymousUser)
	}

	// In a full implementation this would load the User from the database.
	// For now, attach what's stored in the session (set by login()).
	user := &RequestUser{
		ID:              int(userID),
		Username:        stringFrom(session.Get("_auth_user_username")),
		Email:           stringFrom(session.Get("_auth_user_email")),
		IsAuthenticated: true,
		IsStaff:         boolFrom(session.Get("_auth_user_is_staff")),
		IsSuperuser:     boolFrom(session.Get("_auth_user_is_superuser")),
	}
	return context.WithValue(ctx, userKey, user)
}

// UserFrom retrieves the current user from the request context —
// mirrors accessing request.user in a Django view.
func UserFrom(r *http.Request) *RequestUser {
	user, _ := r.Context().Value(userKey).(*RequestUser)
	if user == nil {
		return AnonymousUser
	}
	return user
}

func stringFrom(v interface{}) string {
	s, _ := v.(string)
	return s
}

func boolFrom(v interface{}) bool {
	b, _ := v.(bool)
	return b
}
