// Package conf provides the global settings registry for djanGO projects.
// It mirrors Django's django.conf module.
//
// Usage in your project's settings package (e.g. mysite/mysite/settings.go):
//
//	func init() {
//	    conf.Configure(conf.Settings{
//	        InstalledApps: InstalledApps,
//	        Databases:     Databases,
//	        Debug:         Debug,
//	        SecretKey:     SecretKey,
//	    })
//	}
//
// Management commands (makemigrations, migrate) read from conf.Global.
// This mirrors Django's django.conf.settings object.
package conf

import "sync"

// Settings holds project-level configuration — equivalent to Django's settings module.
type Settings struct {
	InstalledApps []string
	Debug         bool
	SecretKey     string
	AllowedHosts  []string
	Databases     map[string]map[string]string
	StaticURL     string
	TimeZone      string
	LanguageCode  string
}

var (
	mu     sync.RWMutex
	global Settings
	loaded bool
)

// Configure sets the global project settings.
// Call this from your settings package's init() function.
// Equivalent to Django reading settings from DJANGO_SETTINGS_MODULE.
func Configure(s Settings) {
	mu.Lock()
	defer mu.Unlock()
	global = s
	loaded = true
}

// Global returns the current project settings.
// Panics if Configure() has not been called — mirrors Django's
// "django.core.exceptions.ImproperlyConfigured: Requested settings but
// settings are not configured."
func Global() Settings {
	mu.RLock()
	defer mu.RUnlock()
	if !loaded {
		panic("djanGO: settings not configured. Call conf.Configure() from your settings init().")
	}
	return global
}

// IsConfigured returns true if Configure() has been called.
func IsConfigured() bool {
	mu.RLock()
	defer mu.RUnlock()
	return loaded
}

// IsInstalled returns true if the given app label is in InstalledApps.
// Equivalent to Django's apps.is_installed(app_name).
func IsInstalled(appLabel string) bool {
	mu.RLock()
	defer mu.RUnlock()
	if !loaded {
		return false
	}
	for _, a := range global.InstalledApps {
		if a == appLabel {
			return true
		}
	}
	return false
}
