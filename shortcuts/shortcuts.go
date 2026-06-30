// Package shortcuts mirrors Django's django.shortcuts module.
//
// Django:
//
//	from django.shortcuts import render, redirect, get_object_or_404
//
// djanGO:
//
//	import "github.com/mqnifestkelvin/djanGO/shortcuts"
//
//	func MyView(w http.ResponseWriter, r *http.Request) {
//	    shortcuts.Render(w, r, "blog/index.html", shortcuts.Context{"posts": posts})
//	    shortcuts.Redirect(w, r, "blog-index")
//	}
package shortcuts

import (
	"net/http"
	"strings"

	dhttp "github.com/mqnifestkelvin/djanGO/core/http"
	"github.com/mqnifestkelvin/djanGO/core/middleware"
	dtmpl "github.com/mqnifestkelvin/djanGO/core/template"
	"github.com/mqnifestkelvin/djanGO/core/urls"
)

// Context mirrors Django's template context dict — passed to Render().
type Context map[string]interface{}

// Render mirrors Django's render(request, template_name, context, status).
//
// Django:
//
//	return render(request, "blog/index.html", {"posts": posts})
//
// djanGO:
//
//	shortcuts.Render(w, r, "blog/index.html", shortcuts.Context{"posts": posts})
func Render(w http.ResponseWriter, r *http.Request, templateName string, ctx Context, status ...int) {
	code := 200
	if len(status) > 0 {
		code = status[0]
	}

	// Inject request.user into the context — mirrors Django's
	// django.contrib.auth.context_processors.auth which makes
	// {{ user }} available in every template automatically.
	if ctx == nil {
		ctx = Context{}
	}
	if _, hasUser := ctx["user"]; !hasUser {
		ctx["user"] = middleware.UserFrom(r)
	}

	// LoadWithInheritance parses the template alongside base.html so that
	// {{template "base.html" .}} / {{block}} inheritance resolves correctly —
	// mirrors Django's template loader automatically resolving {% extends %}.
	tmpl, err := dtmpl.LoadWithInheritance(templateName, r, "base.html")
	if err != nil {
		http.Error(w, "TemplateDoesNotExist: "+templateName, http.StatusInternalServerError)
		return
	}

	// Execute the "base" template by name so {{block}} overrides from the child
	// are applied — mirrors Django rendering the full {% extends %} chain.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	if err := tmpl.ExecuteTemplate(w, "base", ctx); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// Redirect mirrors Django's redirect(to, permanent=False).
//
// The `to` argument can be:
//   - A named URL:  shortcuts.Redirect(w, r, "blog-index")
//   - A URL path:   shortcuts.Redirect(w, r, "/blog/")
//
// Django:
//
//	return redirect("blog-index")
//	return redirect("/blog/")
func Redirect(w http.ResponseWriter, r *http.Request, to string, permanent ...bool) {
	isPermanent := len(permanent) > 0 && permanent[0]
	location := resolveURL(to)
	if isPermanent {
		dhttp.PermanentRedirect(w, r, location)
	} else {
		dhttp.Redirect(w, r, location)
	}
}

// resolveURL mirrors Django's resolve_url() — tries reverse() first, falls back to the string.
func resolveURL(to string) string {
	// If it already looks like a path, use as-is
	if strings.HasPrefix(to, "/") || strings.HasPrefix(to, "./") || strings.HasPrefix(to, "../") {
		return to
	}
	// Try reverse() — if it panics (no match), fall back to the string as a URL
	resolved, ok := tryReverse(to)
	if ok {
		return resolved
	}
	return to
}

// tryReverse calls urls.Reverse and recovers from the NoReverseMatch panic.
func tryReverse(name string) (path string, ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	return urls.Reverse(name), true
}

// GetObjectOr404 mirrors Django's get_object_or_404().
//
// Django:
//
//	post = get_object_or_404(Post, slug=slug)
//
// djanGO — pass a lookup function that returns (object, error):
//
//	post, err := shortcuts.GetObjectOr404(func() (*Post, error) {
//	    return db.PostBySlug(slug)
//	})
//
// If the lookup returns an error, the handler writes a 404 and returns nil.
// Always check for nil before using the result.
func GetObjectOr404[T any](w http.ResponseWriter, lookup func() (T, error)) (T, bool) {
	obj, err := lookup()
	if err != nil {
		dhttp.NotFound(w)
		var zero T
		return zero, false
	}
	return obj, true
}
