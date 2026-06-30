// Package staticfiles mirrors Django's django.contrib.staticfiles.
//
// Django reference: django/contrib/staticfiles/finders.py
//                   django/contrib/staticfiles/management/commands/collectstatic.py
//                   django/contrib/staticfiles/views.py
//
// Django's static file pipeline:
//  1. Each app can have a static/ directory: blog/static/blog/style.css
//  2. STATICFILES_DIRS adds extra search directories
//  3. `collectstatic` copies all static files to STATIC_ROOT for production
//  4. In development (DEBUG=True), runserver serves them directly from their source dirs
//  5. {% static "blog/style.css" %} generates the URL: STATIC_URL + "blog/style.css"
//
// djanGO implements:
//  - AppDirectoriesFinder — searches <app>/static/ in each InstalledApp
//  - FileSystemFinder     — searches STATICFILES_DIRS
//  - Find(path)           — finds a file across all finders
//  - collectstatic command
//  - Dev serving at STATIC_URL
package staticfiles

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mqnifestkelvin/djanGO/conf"
)

// Find searches all finders for the given relative path —
// mirrors Django's finders.find(path).
//
// Django:
//
//	from django.contrib.staticfiles import finders
//	absolute_path = finders.find("blog/style.css")
//	# → "/home/user/project/blog/static/blog/style.css"
func Find(path string) string {
	path = filepath.FromSlash(strings.TrimLeft(path, "/"))

	// 1. AppDirectoriesFinder — search <app>/static/ in each InstalledApp.
	// Mirrors Django's AppDirectoriesFinder.find().
	for _, dir := range appStaticDirs() {
		full := filepath.Join(dir, path)
		if _, err := os.Stat(full); err == nil {
			return full
		}
	}

	// 2. FileSystemFinder — search STATICFILES_DIRS.
	// Mirrors Django's FileSystemFinder.find().
	for _, dir := range conf.Global().StaticDirs {
		full := filepath.Join(dir, path)
		if _, err := os.Stat(full); err == nil {
			return full
		}
	}

	return ""
}

// AllFiles returns a map of relative path → absolute path for every static file
// found across all finders — used by collectstatic.
//
// Mirrors Django's collectstatic iterating all finders.
func AllFiles() map[string]string {
	result := make(map[string]string)

	// AppDirectoriesFinder.
	for _, dir := range appStaticDirs() {
		_ = filepath.Walk(dir, func(abs string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(dir, abs)
			if _, exists := result[rel]; !exists {
				result[rel] = abs
			}
			return nil
		})
	}

	// FileSystemFinder.
	for _, dir := range conf.Global().StaticDirs {
		_ = filepath.Walk(dir, func(abs string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(dir, abs)
			if _, exists := result[rel]; !exists {
				result[rel] = abs
			}
			return nil
		})
	}

	return result
}

// appStaticDirs returns the list of <app>/static/ directories that exist.
// Mirrors Django's AppDirectoriesFinder building its locations list.
func appStaticDirs() []string {
	cwd, _ := os.Getwd()
	var dirs []string
	for _, app := range conf.Global().InstalledApps {
		parts := strings.Split(app, ".")
		label := parts[len(parts)-1]
		candidate := filepath.Join(cwd, label, "static")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			dirs = append(dirs, candidate)
		}
	}
	return dirs
}
