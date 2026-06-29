// Package cors mirrors django-cors-headers (corsheaders).
//
// Django reference: django-cors-headers/corsheaders/middleware.py
//                   django-cors-headers/corsheaders/conf.py
//
// Usage — in your settings.go:
//
//	InstalledApps: []string{
//	    "djanGO.contrib.cors",
//	    ...
//	},
//	Middleware: []string{
//	    "djanGO.contrib.cors.middleware.CorsMiddleware",  // must be FIRST
//	    ...
//	},
//	CorsAllowAllOrigins: true,
//	// or:
//	CorsAllowedOrigins: []string{"https://example.com"},
//
// Django equivalent:
//
//	INSTALLED_APPS = ["corsheaders", ...]
//	MIDDLEWARE = ["corsheaders.middleware.CorsMiddleware", ...]
//	CORS_ALLOW_ALL_ORIGINS = True
//	# or:
//	CORS_ALLOWED_ORIGINS = ["https://example.com"]
package cors

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/mqnifestkelvin/djanGO/conf"
	"github.com/mqnifestkelvin/djanGO/core/middleware"
)

// Default allowed headers — mirrors corsheaders.defaults.default_headers.
var defaultAllowHeaders = []string{
	"accept",
	"authorization",
	"content-type",
	"user-agent",
	"x-csrftoken",
	"x-requested-with",
}

// Default allowed methods — mirrors corsheaders.defaults.default_methods.
var defaultAllowMethods = []string{
	"DELETE", "GET", "OPTIONS", "PATCH", "POST", "PUT",
}

func init() {
	middleware.Register("djanGO.contrib.cors.middleware.CorsMiddleware", CorsMiddleware)
}

// CorsMiddleware mirrors Django's corsheaders.middleware.CorsMiddleware.
//
// Django:
//
//	class CorsMiddleware:
//	    def __call__(self, request):
//	        response = self.check_preflight(request)
//	        if response is None:
//	            response = self.get_response(request)
//	        self.add_response_headers(request, response)
//	        return response
func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// No Origin header means not a cross-origin request — pass through.
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		s := corsSettings()

		// Check if CORS is enabled for this URL path.
		// Mirrors: re.match(conf.CORS_URLS_REGEX, request.path_info)
		if !pathAllowed(r.URL.Path, s.urlsRegex) {
			next.ServeHTTP(w, r)
			return
		}

		// Preflight (OPTIONS) request — mirrors CorsMiddleware.check_preflight().
		// Django returns an empty 200 response for valid preflight requests.
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			if originAllowed(origin, s) {
				addCorsHeaders(w, r, origin, s, true)
				w.Header().Set("Content-Length", "0")
				w.WriteHeader(http.StatusOK)
			}
			return
		}

		// Regular request — add CORS headers then serve.
		// Vary: Origin added only when CORS is active so caches distinguish
		// responses by origin — mirrors Django's patch_vary_headers(response, ("origin",)).
		if originAllowed(origin, s) {
			w.Header().Add("Vary", "Origin")
			addCorsHeaders(w, r, origin, s, false)
		}
		next.ServeHTTP(w, r)
	})
}

type corsConfig struct {
	allowAllOrigins    bool
	allowCredentials   bool
	allowPrivateNetwork bool
	allowedOrigins     []string
	allowedOriginsRe   []string
	allowHeaders       []string
	allowMethods       []string
	exposeHeaders      []string
	preflightMaxAge    int
	urlsRegex          string
}

func corsSettings() corsConfig {
	if !conf.IsConfigured() {
		return corsConfig{urlsRegex: `^.*$`, allowHeaders: defaultAllowHeaders, allowMethods: defaultAllowMethods, preflightMaxAge: 86400}
	}
	s := conf.Global()
	c := corsConfig{
		allowAllOrigins:    s.CorsAllowAllOrigins,
		allowCredentials:   s.CorsAllowCredentials,
		allowPrivateNetwork: s.CorsAllowPrivateNetwork,
		allowedOrigins:     s.CorsAllowedOrigins,
		allowedOriginsRe:   s.CorsAllowedOriginRegexes,
		allowHeaders:       s.CorsAllowHeaders,
		allowMethods:       s.CorsAllowMethods,
		exposeHeaders:      s.CorsExposeHeaders,
		preflightMaxAge:    s.CorsPreflightMaxAge,
		urlsRegex:          s.CorsURLsRegex,
	}
	if len(c.allowHeaders) == 0 {
		c.allowHeaders = defaultAllowHeaders
	}
	if len(c.allowMethods) == 0 {
		c.allowMethods = defaultAllowMethods
	}
	if c.preflightMaxAge == 0 {
		c.preflightMaxAge = 86400
	}
	if c.urlsRegex == "" {
		c.urlsRegex = `^.*$`
	}
	return c
}

func pathAllowed(path, pattern string) bool {
	matched, err := regexp.MatchString(pattern, path)
	return err == nil && matched
}

// originAllowed checks CORS_ALLOW_ALL_ORIGINS, CORS_ALLOWED_ORIGINS, and
// CORS_ALLOWED_ORIGIN_REGEXES — mirrors CorsMiddleware.origin_found_in_white_lists().
func originAllowed(origin string, s corsConfig) bool {
	if s.allowAllOrigins {
		return true
	}
	for _, o := range s.allowedOrigins {
		if o == origin {
			return true
		}
	}
	for _, pattern := range s.allowedOriginsRe {
		if matched, _ := regexp.MatchString(pattern, origin); matched {
			return true
		}
	}
	return false
}

// addCorsHeaders sets the CORS response headers —
// mirrors CorsMiddleware.add_response_headers().
//
// Django:
//
//	response["access-control-allow-origin"] = "*"  # or origin
//	response["access-control-allow-credentials"] = "true"
//	response["access-control-allow-headers"] = ", ".join(conf.CORS_ALLOW_HEADERS)
//	response["access-control-allow-methods"] = ", ".join(conf.CORS_ALLOW_METHODS)
func addCorsHeaders(w http.ResponseWriter, r *http.Request, origin string, s corsConfig, preflight bool) {
	h := w.Header()

	// Access-Control-Allow-Origin
	if s.allowAllOrigins && !s.allowCredentials {
		h.Set("Access-Control-Allow-Origin", "*")
	} else {
		h.Set("Access-Control-Allow-Origin", origin)
	}

	// Access-Control-Allow-Credentials
	if s.allowCredentials {
		h.Set("Access-Control-Allow-Credentials", "true")
	}

	// Access-Control-Expose-Headers (simple requests)
	if len(s.exposeHeaders) > 0 {
		h.Set("Access-Control-Expose-Headers", strings.Join(s.exposeHeaders, ", "))
	}

	// Preflight-only headers
	if preflight {
		h.Set("Access-Control-Allow-Headers", strings.Join(s.allowHeaders, ", "))
		h.Set("Access-Control-Allow-Methods", strings.Join(s.allowMethods, ", "))
		if s.preflightMaxAge > 0 {
			h.Set("Access-Control-Max-Age", itoa(s.preflightMaxAge))
		}
	}

	// Private Network Access
	if s.allowPrivateNetwork && r.Header.Get("Access-Control-Request-Private-Network") == "true" {
		h.Set("Access-Control-Allow-Private-Network", "true")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}
