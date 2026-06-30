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

import (
	"sync"

	"github.com/mqnifestkelvin/djanGO/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

// Settings holds project-level configuration — equivalent to Django's settings module.
type Settings struct {
	InstalledApps []string
	Middleware     []string
	Debug         bool
	SecretKey     string
	AllowedHosts  []string
	Databases     map[string]map[string]string
	StaticURL     string   // URL prefix for static files, e.g. "/static/"  (STATIC_URL)
	StaticRoot    string   // Absolute path where collectstatic copies files  (STATIC_ROOT)
	StaticDirs    []string // Additional directories to search for static files (STATICFILES_DIRS)
	TimeZone      string
	LanguageCode  string

	// CORS settings — mirrors django-cors-headers configuration.
	// Django: CORS_ALLOW_ALL_ORIGINS, CORS_ALLOWED_ORIGINS, etc.
	//
	// Usage:
	//   CorsAllowAllOrigins: true                          // CORS_ALLOW_ALL_ORIGINS
	//   CorsAllowedOrigins: []string{"https://foo.com"}   // CORS_ALLOWED_ORIGINS
	//   CorsAllowCredentials: true                        // CORS_ALLOW_CREDENTIALS
	//   CorsAllowedOriginRegexes: []string{`^https://.*\.example\.com$`}
	CorsAllowAllOrigins      bool
	CorsAllowedOrigins       []string
	CorsAllowedOriginRegexes []string
	CorsAllowCredentials     bool
	CorsAllowPrivateNetwork  bool
	CorsAllowHeaders         []string
	CorsAllowMethods         []string
	CorsExposeHeaders        []string
	CorsPreflightMaxAge      int
	CorsURLsRegex            string
}

var (
	mu     sync.RWMutex
	global Settings
	loaded bool
)

// Configure sets the global project settings and wires up the database —
// mirrors Django reading DATABASES from settings and calling django.db.setup().
// Call this from your settings package's init() function.
func Configure(s Settings) {
	mu.Lock()
	defer mu.Unlock()
	global = s
	loaded = true

	// Register each database alias with Beego's ORM —
	// mirrors Django's DATABASES setting wiring into django.db.connections.
	//
	// Django:
	//   DATABASES = {"default": {"ENGINE": "sqlite3", "NAME": "db.sqlite3"}}
	//
	// djanGO: same structure, we register each alias automatically.
	for alias, db := range s.Databases {
		engine := db["ENGINE"]
		name := db["NAME"]
		if engine == "" || name == "" {
			continue
		}
		// Map Django ENGINE names to Beego driver names
		driver := engineToDriver(engine)
		dsn := buildDSN(driver, name, db)
		// Ignore re-registration errors (init() may run more than once in tests)
		_ = orm.RegisterDataBase(alias, driver, dsn)
	}
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

// engineToDriver maps Django ENGINE names to Beego/Go driver names.
func engineToDriver(engine string) string {
	switch engine {
	case "sqlite3", "django.db.backends.sqlite3":
		return "sqlite3"
	case "mysql", "django.db.backends.mysql":
		return "mysql"
	case "postgres", "postgresql", "django.db.backends.postgresql":
		return "postgres"
	default:
		return engine
	}
}

// buildDSN constructs the driver DSN from a DATABASES entry.
func buildDSN(driver, name string, db map[string]string) string {
	switch driver {
	case "sqlite3":
		return name
	case "mysql":
		user := db["USER"]
		pass := db["PASSWORD"]
		host := db["HOST"]
		if host == "" {
			host = "127.0.0.1"
		}
		port := db["PORT"]
		if port == "" {
			port = "3306"
		}
		return user + ":" + pass + "@tcp(" + host + ":" + port + ")/" + name + "?charset=utf8mb4"
	case "postgres":
		user := db["USER"]
		pass := db["PASSWORD"]
		host := db["HOST"]
		if host == "" {
			host = "127.0.0.1"
		}
		port := db["PORT"]
		if port == "" {
			port = "5432"
		}
		return "user=" + user + " password=" + pass + " host=" + host + " port=" + port + " dbname=" + name + " sslmode=disable"
	default:
		return name
	}
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
