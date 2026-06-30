// Package urls mirrors Django's django.urls module.
//
// Django usage:
//
//	from django.urls import path, include
//
//	urlpatterns = [
//	    path('blog/', include('blog.urls')),
//	    path('about/', views.about, name='about'),
//	]
//
// djanGO usage:
//
//	import "github.com/mqnifestkelvin/djanGO/core/urls"
//
//	var URLPatterns = urls.Patterns(
//	    urls.Path("/blog/", blog.URLPatterns),
//	    urls.Path("/about/", AboutView, "about"),
//	)
package urls

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// ViewFunc is a standard Go HTTP handler — Django's equivalent of a view function.
type ViewFunc func(http.ResponseWriter, *http.Request)

// URLPattern is a single URL rule — mirrors Django's URLPattern / URLResolver.
type URLPattern struct {
	prefix   string     // URL prefix for this pattern
	view     ViewFunc   // nil if this is an include()
	included []*URLPattern // child patterns from include()
	name     string     // named URL for reverse()
}

// registry holds all named patterns for reverse().
var (
	mu     sync.RWMutex
	byName = make(map[string]string) // name → full resolved path
)

// Path registers a URL pattern. Mirrors Django's path().
//
// Two forms:
//
//	urls.Path("/about/", AboutView, "about")          // view with optional name
//	urls.Path("/blog/", blog.URLPatterns)             // include another app's patterns
func Path(prefix string, target interface{}, name ...string) *URLPattern {
	p := &URLPattern{prefix: prefix}

	switch t := target.(type) {
	case ViewFunc:
		p.view = t
	case func(http.ResponseWriter, *http.Request):
		p.view = ViewFunc(t)
	case []*URLPattern:
		p.included = t
	default:
		panic(fmt.Sprintf("urls.Path: unsupported target type %T — must be ViewFunc or []*URLPattern", target))
	}

	if len(name) > 0 && name[0] != "" {
		p.name = name[0]
	}

	return p
}

// Include returns a child URLPattern slice — mirrors Django's include().
// Use inside Path() to mount an app's URLPatterns under a prefix.
func Include(patterns []*URLPattern) []*URLPattern {
	return patterns
}

// Patterns collects a list of URLPattern entries — mirrors Django's urlpatterns list.
// Assign the result to a package-level var named URLPatterns in each app's urls.go.
func Patterns(ps ...*URLPattern) []*URLPattern {
	return ps
}

// Reverse resolves a named URL to its path — mirrors Django's reverse().
// Panics if the name is not registered (same as Django's NoReverseMatch).
func Reverse(name string, args ...string) string {
	mu.RLock()
	path, ok := byName[name]
	mu.RUnlock()
	if !ok {
		panic(fmt.Sprintf("urls.Reverse: no URL pattern with name '%s'", name))
	}
	for _, arg := range args {
		// Replace the first {param} placeholder with the arg
		start := strings.Index(path, "{")
		end := strings.Index(path, "}")
		if start >= 0 && end > start {
			path = path[:start] + arg + path[end+1:]
		}
	}
	return path
}

// rootPatterns holds the project-level URLPatterns registered via Register().
var rootPatterns []*URLPattern

// Register sets the root URL patterns for the project — mirrors Django's ROOT_URLCONF.
// Call this from your project's urls.go init() with your top-level URLPatterns.
//
//	func init() {
//	    urls.Register(URLPatterns)
//	}
func Register(patterns []*URLPattern) {
	mu.Lock()
	rootPatterns = patterns
	mu.Unlock()
}

// Registered returns the root URL patterns set via Register().
func Registered() []*URLPattern {
	mu.RLock()
	defer mu.RUnlock()
	return rootPatterns
}

// Mount registers all patterns from a []*URLPattern slice into the HTTP mux.
// Called once at server startup — mirrors Django's URL resolver walking urlpatterns.
// Also registers named patterns into the reverse() registry with their full paths.
func Mount(mux *http.ServeMux, patterns []*URLPattern, prefix string) {
	for _, p := range patterns {
		fullPath := joinPath(prefix, p.prefix)
		if len(p.included) > 0 {
			Mount(mux, p.included, fullPath)
		} else if p.view != nil {
			if p.name != "" {
				mu.Lock()
				byName[p.name] = fullPath
				mu.Unlock()
			}
			mux.HandleFunc(fullPath, http.HandlerFunc(p.view))
		}
	}
}

// joinPath joins a prefix and suffix into a clean URL path.
func joinPath(prefix, suffix string) string {
	if suffix == "/" || suffix == "" {
		return strings.TrimRight(prefix, "/") + "/"
	}
	return strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(suffix, "/")
}
