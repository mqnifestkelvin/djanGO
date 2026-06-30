// Package template mirrors Django's template engine.
//
// Django's APP_DIRS loader walks every installed app looking for a
// "templates/" subdirectory and adds it to the search path.  We do the
// same: given the project working directory and the list of InstalledApps,
// we build a search path of  <cwd>/<appLabel>/templates/  directories.
//
// Django:
//
//	TEMPLATES = [{"BACKEND": "...", "APP_DIRS": True, ...}]
//	# then in a view:
//	return render(request, "blog/index.html", context)
//
// djanGO:
//
//	shortcuts.Render(w, r, "blog/index.html", shortcuts.Context{...})
//	# loader finds it in  blog/templates/blog/index.html
package template

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mqnifestkelvin/djanGO/conf"
	"github.com/mqnifestkelvin/djanGO/core/urls"
)

var (
	mu    sync.RWMutex
	cache = make(map[string]*template.Template)
)

// templateDirs returns the ordered list of template search directories —
// mirrors Django's get_app_template_dirs("templates").
//
// For each app label in InstalledApps we look for:
//
//	<cwd>/<appLabel>/templates/
//
// Only directories that actually exist are included.
func templateDirs() []string {
	cwd, _ := os.Getwd()
	var dirs []string

	apps := conf.Global().InstalledApps
	for _, app := range apps {
		// Only use the last path segment as the directory name —
		// "djanGO.contrib.admin" → skip (no local dir), "blog" → "./blog/templates/"
		parts := strings.Split(app, ".")
		label := parts[len(parts)-1]
		candidate := filepath.Join(cwd, label, "templates")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			dirs = append(dirs, candidate)
		}
	}
	return dirs
}

// findTemplate searches each template directory for the named template —
// mirrors Django's FilesystemLoader.get_template_sources().
func findTemplate(name string) (string, error) {
	for _, dir := range templateDirs() {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if _, err := os.Stat(full); err == nil {
			return full, nil
		}
	}
	return "", fmt.Errorf("template %q not found in any app templates/ directory", name)
}

// csrfTokenFunc is set by the middleware package to avoid an import cycle.
// middleware → template would be circular; instead middleware registers its getter here.
var csrfTokenFunc func(r *http.Request) string

// SetCSRFTokenFunc registers the CSRF token getter — called by middleware init().
func SetCSRFTokenFunc(fn func(r *http.Request) string) {
	csrfTokenFunc = fn
}

// funcMap provides template functions available in every template —
// mirrors Django's built-in template tags.
//
// Django:   {% url "blog-index" %}
// djanGO:   {{ url "blog-index" }}
//
// Django:   {% static "blog/style.css" %}
// djanGO:   {{ static "blog/style.css" }}
//
// Django:   {% csrf_token %}
// djanGO:   {{ csrf_token }}   (returns the hidden input HTML)
func funcMap(r *http.Request) template.FuncMap {
	staticURL := conf.Global().StaticURL
	return template.FuncMap{
		// url mirrors Django's {% url "name" args... %}
		"url": func(name string, args ...string) (string, error) {
			defer func() {}()
			resolved, ok := tryReverse(name, args...)
			if !ok {
				return "", fmt.Errorf("url: no pattern named %q", name)
			}
			return resolved, nil
		},
		// static mirrors Django's {% static "path" %}
		"static": func(path string) string {
			base := strings.TrimRight(staticURL, "/")
			return base + "/" + strings.TrimLeft(path, "/")
		},
		// csrf_token mirrors Django's {% csrf_token %} —
		// renders <input type="hidden" name="csrfmiddlewaretoken" value="...">
		"csrf_token": func() template.HTML {
			token := ""
			if csrfTokenFunc != nil && r != nil {
				token = csrfTokenFunc(r)
			}
			if token == "" {
				return ""
			}
			return template.HTML(`<input type="hidden" name="csrfmiddlewaretoken" value="` + token + `">`)
		},
	}
}

func tryReverse(name string, args ...string) (path string, ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	return urls.Reverse(name, args...), true
}

// Load finds, parses, and returns a template by name —
// mirrors Django's template.loader.get_template(name).
// r is used to populate request-specific template functions (csrf_token).
func Load(name string, r *http.Request) (*template.Template, error) {
	path, err := findTemplate(name)
	if err != nil {
		return nil, err
	}
	// Templates are NOT cached here because funcMap is request-specific (csrf_token).
	t, err := template.New(filepath.Base(path)).Funcs(funcMap(r)).ParseFiles(path)
	if err != nil {
		return nil, fmt.Errorf("template %q: %w", name, err)
	}
	return t, nil
}

// LoadWithInheritance loads a template that uses {% extends %} —
// parses the named template plus all base templates so block inheritance resolves.
//
// r is used to populate request-specific template functions (csrf_token).
func LoadWithInheritance(name string, r *http.Request, extra ...string) (*template.Template, error) {
	primary, err := findTemplate(name)
	if err != nil {
		return nil, err
	}

	// Parse bases first, child last — in Go, the LAST {{define}} for a given
	// name wins, so child definitions override base defaults.
	files := []string{}
	for _, e := range extra {
		if p, err := findTemplate(e); err == nil {
			files = append(files, p)
		}
	}
	files = append(files, primary)

	t, err := template.New(filepath.Base(primary)).Funcs(funcMap(r)).ParseFiles(files...)
	if err != nil {
		return nil, fmt.Errorf("template %q: %w", name, err)
	}
	return t, nil
}

// ClearCache flushes the template cache — useful in development.
func ClearCache() {
	mu.Lock()
	cache = make(map[string]*template.Template)
	mu.Unlock()
}
