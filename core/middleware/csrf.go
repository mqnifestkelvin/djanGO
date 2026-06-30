package middleware

// CSRF token validation — mirrors Django's CsrfViewMiddleware.
//
// Django reference: django/middleware/csrf.py
//
// How Django CSRF works:
//  1. On first request, generate a 32-char random secret, store in cookie "csrftoken"
//  2. Template tag {% csrf_token %} embeds a masked token in the form
//  3. On POST/PUT/PATCH/DELETE: read the token from X-CSRFToken header or form field "csrfmiddlewaretoken"
//  4. Unmask the token and compare its secret to the cookie secret
//  5. If they don't match → 403 Forbidden
//
// Safe methods (GET, HEAD, OPTIONS, TRACE) always pass through.
//
// djanGO simplification:
//   - We store the raw 32-char secret in both the cookie and session (no masking for simplicity)
//   - Token in form: csrfmiddlewaretoken
//   - Token in header: X-CSRFToken
//   - @csrf_exempt decorator skips validation for a specific view

import (
	"context"
	"crypto/rand"
	"math/big"
	"net/http"
	"strings"

	"github.com/mqnifestkelvin/djanGO/core/template"
	"github.com/mqnifestkelvin/djanGO/core/urls"
)

func init() {
	// Register the CSRF token getter with the template package.
	// Done here to avoid an import cycle: template cannot import middleware.
	template.SetCSRFTokenFunc(CSRFTokenFromContext)
}

const (
	csrfCookieName  = "csrftoken"
	csrfHeaderName  = "X-CSRFToken"
	csrfFieldName   = "csrfmiddlewaretoken"
	csrfTokenLength = 32
	csrfAllowedChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type csrfExemptKey struct{}

// CsrfExempt marks a view as exempt from CSRF validation —
// mirrors Django's @csrf_exempt decorator.
//
// Django:
//
//	from django.views.decorators.csrf import csrf_exempt
//
//	@csrf_exempt
//	def my_api_view(request): ...
func CsrfExempt(view urls.ViewFunc) urls.ViewFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), csrfExemptKey{}, true)
		view(w, r.WithContext(ctx))
	}
}

// GetCSRFToken returns the current CSRF token for use in templates —
// mirrors Django's get_token(request).
//
// Django:
//
//	from django.middleware.csrf import get_token
//	token = get_token(request)  # used by {% csrf_token %} template tag
func GetCSRFToken(r *http.Request) string {
	// Try cookie first.
	if cookie, err := r.Cookie(csrfCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return ""
}

// newCSRFToken generates a new random 32-char CSRF secret —
// mirrors Django's _get_new_csrf_string().
func newCSRFToken() string {
	b := make([]byte, csrfTokenLength)
	chars := []byte(csrfAllowedChars)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b[i] = chars[n.Int64()]
	}
	return string(b)
}

// csrfMiddleware is the full implementation called from the registered CSRFMiddleware.
// Split from the registration stub in builtin.go.
func csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for @csrf_exempt.
		if exempt, _ := r.Context().Value(csrfExemptKey{}).(bool); exempt {
			next.ServeHTTP(w, r)
			return
		}

		// Safe methods always pass through — mirrors Django's safe method check.
		safeMethods := map[string]bool{
			"GET": true, "HEAD": true, "OPTIONS": true, "TRACE": true,
		}
		if safeMethods[r.Method] {
			// Ensure cookie is set for future POST requests.
			ensureCSRFCookie(w, r)
			next.ServeHTTP(w, r)
			return
		}

		// Get the secret from the cookie.
		cookie, err := r.Cookie(csrfCookieName)
		if err != nil || cookie.Value == "" {
			http.Error(w, "Forbidden (CSRF cookie not set)", http.StatusForbidden)
			return
		}
		cookieSecret := cookie.Value

		// Get the token from the request — header takes precedence over form field,
		// mirrors Django's CsrfViewMiddleware._get_token().
		requestToken := r.Header.Get(csrfHeaderName)
		if requestToken == "" {
			_ = r.ParseForm()
			requestToken = r.FormValue(csrfFieldName)
		}

		if requestToken == "" {
			http.Error(w, "Forbidden (CSRF token missing)", http.StatusForbidden)
			return
		}

		// Compare: both must match (constant-time to prevent timing attacks).
		if !constantTimeEqual(cookieSecret, requestToken) {
			http.Error(w, "Forbidden (CSRF token invalid)", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ensureCSRFCookie sets the csrftoken cookie if not already present.
// Mirrors Django's CsrfViewMiddleware.process_response setting the cookie.
func ensureCSRFCookie(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie(csrfCookieName); err == nil {
		return // already set
	}
	token := newCSRFToken()
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		// Not HttpOnly — JS needs to read it to put in X-CSRFToken header.
		// Mirrors Django's CSRF_COOKIE_HTTPONLY=False default.
	})
}

// constantTimeEqual compares two strings in constant time to prevent timing attacks —
// mirrors Django's constant_time_compare().
func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := 0; i < len(a); i++ {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

// CSRFTokenFromContext is a template helper — returns the CSRF token for the current request.
// Used by the {% csrf_token %} template tag.
func CSRFTokenFromContext(r *http.Request) string {
	token := GetCSRFToken(r)
	if token == "" {
		token = newCSRFToken()
	}
	return token
}

// isSafeMethod returns true for HTTP methods that don't modify state.
func isSafeMethod(method string) bool {
	return strings.ToUpper(method) == "GET" ||
		strings.ToUpper(method) == "HEAD" ||
		strings.ToUpper(method) == "OPTIONS" ||
		strings.ToUpper(method) == "TRACE"
}
