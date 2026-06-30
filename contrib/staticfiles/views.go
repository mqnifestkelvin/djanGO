package staticfiles

// Dev static file serving — mirrors Django's staticfiles.views.serve().
//
// Django:
//
//	# Django's runserver auto-serves static files in DEBUG mode:
//	# urls.py automatically gets: path("static/<path:path>", serve)
//
//	from django.contrib.staticfiles.views import serve
//
// djanGO: StaticFileHandler() returns an http.Handler that serves files
// found by the finders. Mount it at STATIC_URL in runserver.

import (
	"net/http"
	"strings"

	"github.com/mqnifestkelvin/djanGO/conf"
)

// Handler returns an http.Handler that serves static files in development —
// mirrors Django's staticfiles view server (DEBUG mode only).
//
// Django:
//
//	# Automatically active when DEBUG=True and django.contrib.staticfiles is installed.
//	# Serves files from each app's static/ directory and STATICFILES_DIRS.
//
// Mount at STATIC_URL:
//
//	mux.Handle("/static/", staticfiles.Handler())
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		staticURL := conf.Global().StaticURL
		if staticURL == "" {
			staticURL = "/static/"
		}

		// Strip the STATIC_URL prefix to get the relative file path.
		// e.g. /static/blog/style.css → blog/style.css
		path := strings.TrimPrefix(r.URL.Path, staticURL)
		path = strings.TrimPrefix(path, "/")

		if path == "" {
			http.NotFound(w, r)
			return
		}

		abs := Find(path)
		if abs == "" {
			http.NotFound(w, r)
			return
		}

		http.ServeFile(w, r, abs)
	})
}
