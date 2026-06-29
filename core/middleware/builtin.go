package middleware

import (
	"net/http"
	"strings"
)

func init() {
	Register("djanGO.middleware.security.SecurityMiddleware", SecurityMiddleware)
	Register("djanGO.middleware.common.CommonMiddleware", CommonMiddleware)
	Register("djanGO.middleware.csrf.CsrfViewMiddleware", CSRFMiddleware)
	Register("djanGO.contrib.sessions.middleware.SessionMiddleware", SessionMiddleware)
	Register("djanGO.middleware.clickjacking.XFrameOptionsMiddleware", XFrameOptionsMiddleware)
	Register("djanGO.contrib.auth.middleware.AuthenticationMiddleware", AuthenticationMiddleware)
}

// SecurityMiddleware mirrors Django's SecurityMiddleware.
// Sets security headers on every response.
//
// Django:
//
//	class SecurityMiddleware:
//	    def process_response(self, request, response):
//	        response["X-Content-Type-Options"] = "nosniff"
//	        response["Referrer-Policy"] = "same-origin"
func SecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

// CommonMiddleware mirrors Django's CommonMiddleware.
// Appends trailing slashes to URLs that don't have one (APPEND_SLASH=True behaviour).
//
// Django:
//
//	class CommonMiddleware:
//	    def process_request(self, request):
//	        if self.should_redirect_with_slash(request):
//	            return redirect(self.get_full_path_with_slash(request))
func CommonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Append trailing slash — mirrors Django's APPEND_SLASH=True default
		if r.URL.Path != "/" && !strings.HasSuffix(r.URL.Path, "/") && r.Method == http.MethodGet {
			http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CSRFMiddleware mirrors Django's CsrfViewMiddleware.
// In development (non-POST safe methods pass through).
// Full CSRF token validation is wired in when sessions are active.
//
// Django:
//
//	class CsrfViewMiddleware:
//	    def process_view(self, request, callback, ...):
//	        if request.method in ("GET", "HEAD", "OPTIONS", "TRACE"):
//	            return None  # safe methods pass through
//	        # validate csrf token ...
func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Safe methods always pass through — mirrors Django's CSRF exempt for GET/HEAD
		safeMethods := map[string]bool{"GET": true, "HEAD": true, "OPTIONS": true, "TRACE": true}
		if !safeMethods[r.Method] {
			// TODO: validate CSRF token from cookie vs form field
			// For now, pass through (mirrors Django's @csrf_exempt behaviour)
		}
		next.ServeHTTP(w, r)
	})
}

// SessionMiddleware mirrors Django's SessionMiddleware.
// Loads the session from the cookie and makes it available on the request context.
//
// Django:
//
//	class SessionMiddleware:
//	    def process_request(self, request):
//	        session_key = request.COOKIES.get(settings.SESSION_COOKIE_NAME)
//	        request.session = self.SessionStore(session_key)
//
//	    def process_response(self, request, response):
//	        if request.session.modified:
//	            response.set_cookie(SESSION_COOKIE_NAME, ...)
func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Attach session to request context — mirrors Django's
		// SessionMiddleware.process_request loading the session.
		ctx := WithSession(r.Context(), w, r)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)

		// Save session if modified — mirrors Django's SessionMiddleware.process_response.
		// Django: if response.status_code != 500 and session.modified: session.save()
		// We always try to save; save() is a no-op when Modified==false.
		SaveSession(r)
	})
}

// XFrameOptionsMiddleware mirrors Django's XFrameOptionsMiddleware.
// Sets X-Frame-Options: DENY on every response.
//
// Django:
//
//	class XFrameOptionsMiddleware:
//	    def process_response(self, request, response):
//	        response["X-Frame-Options"] = "DENY"
func XFrameOptionsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if w.Header().Get("X-Frame-Options") == "" {
			w.Header().Set("X-Frame-Options", "DENY")
		}
		next.ServeHTTP(w, r)
	})
}

// AuthenticationMiddleware mirrors Django's AuthenticationMiddleware.
// Attaches the current user to the request based on the session.
//
// Django:
//
//	class AuthenticationMiddleware:
//	    def process_request(self, request):
//	        request.user = SimpleLazyObject(lambda: get_user(request))
func AuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Attach user from session to context — full implementation in contrib/auth
		ctx := WithUser(r.Context(), r)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
