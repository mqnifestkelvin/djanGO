package apps

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"
)

// AppConfig holds configuration and metadata for a single djanGO app.
// Every app registers itself via Register() in its init() function.
// Equivalent to Django's AppConfig class in django/apps/config.py.
type AppConfig struct {
	// Name is the unique app identifier, e.g. "blog", "accounts".
	Name string

	// Label is a short name used in migrations and admin. Defaults to Name.
	Label string

	// VerboseName is the human-readable app name shown in admin.
	VerboseName string

	// Path is the filesystem path to the app directory.
	Path string

	// Models lists model names registered by this app.
	Models []string

	// ready is called once after all apps are loaded.
	readyFn func()
}

// Ready is called once all InstalledApps have been loaded.
// Override by setting ReadyFn before registering.
func (c *AppConfig) Ready() {
	if c.readyFn != nil {
		c.readyFn()
	}
}

// SetReady sets the function to call when the app is ready.
func (c *AppConfig) SetReady(fn func()) {
	c.readyFn = fn
}

// registry is the global app registry — equivalent to Django's Apps class.
var registry = &appRegistry{
	configs: make(map[string]*AppConfig),
}

type appRegistry struct {
	mu      sync.RWMutex
	configs map[string]*AppConfig // keyed by app Name
	ordered []string              // insertion order, preserved like Django
	ready   bool
}

// Register adds an AppConfig to the global registry.
// Call this from your app's init() in apps.go.
func Register(cfg *AppConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("djanGO: AppConfig must have a Name")
	}
	if cfg.Label == "" {
		cfg.Label = cfg.Name
	}
	if cfg.VerboseName == "" {
		cfg.VerboseName = cfg.Label
	}
	if cfg.Path == "" {
		cfg.Path = resolveAppPath(cfg.Name)
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	if _, exists := registry.configs[cfg.Name]; exists {
		return fmt.Errorf("djanGO: app '%s' is already registered", cfg.Name)
	}

	registry.configs[cfg.Name] = cfg
	registry.ordered = append(registry.ordered, cfg.Name)
	return nil
}

// MustRegister is like Register but panics on error.
func MustRegister(cfg *AppConfig) {
	if err := Register(cfg); err != nil {
		panic(err)
	}
}

// GetAppConfigs returns all registered AppConfigs in registration order.
// Equivalent to Django's apps.get_app_configs().
func GetAppConfigs() []*AppConfig {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	out := make([]*AppConfig, 0, len(registry.ordered))
	for _, name := range registry.ordered {
		out = append(out, registry.configs[name])
	}
	return out
}

// GetAppConfig returns the AppConfig for a given app name.
// Equivalent to Django's apps.get_app_config(app_label).
func GetAppConfig(name string) (*AppConfig, error) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	cfg, ok := registry.configs[name]
	if !ok {
		return nil, fmt.Errorf("djanGO: no app with name '%s' is installed", name)
	}
	return cfg, nil
}

// IsInstalled reports whether an app with the given name is registered.
// Equivalent to Django's apps.is_installed(app_name).
func IsInstalled(name string) bool {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	_, ok := registry.configs[name]
	return ok
}

// Setup iterates all registered apps and calls Ready() on each.
// Call this once at server startup after all apps have been imported.
// Equivalent to Django's django.setup().
func Setup(installedApps []string) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if registry.ready {
		return nil
	}

	for _, name := range installedApps {
		if _, ok := registry.configs[name]; !ok {
			return fmt.Errorf("djanGO: '%s' is in InstalledApps but was never registered — did you import the app package?", name)
		}
	}

	for _, name := range installedApps {
		registry.configs[name].Ready()
	}

	registry.ready = true
	return nil
}

// AppNames returns the names of all registered apps in order.
func AppNames() []string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	out := make([]string, len(registry.ordered))
	copy(out, registry.ordered)
	return out
}

// resolveAppPath tries to find the app directory on disk by walking
// common project layouts (apps/<name>, <name>/).
func resolveAppPath(name string) string {
	cwd, _ := os.Getwd()

	candidates := []string{
		filepath.Join(cwd, "apps", name),
		filepath.Join(cwd, name),
	}
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	return filepath.Join(cwd, name)
}

// Ensure plugin import is used (needed if we later support .so app plugins).
var _ = plugin.Open
