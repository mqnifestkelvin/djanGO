package apps

import (
	"strings"
	"testing"
)

// Each test uses unique app names to avoid conflicts with the global registry.

func TestRegisterApp(t *testing.T) {
	err := Register(&AppConfig{Name: "t_register"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !IsInstalled("t_register") {
		t.Fatal("expected app to be installed after Register()")
	}
}

func TestRegisterDuplicateApp(t *testing.T) {
	Register(&AppConfig{Name: "t_duplicate"})
	err := Register(&AppConfig{Name: "t_duplicate"})
	if err == nil {
		t.Fatal("expected error on duplicate registration, got nil")
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("expected 'already registered' in error, got: %v", err)
	}
}

func TestMustRegisterPanics(t *testing.T) {
	Register(&AppConfig{Name: "t_must_panic"})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected MustRegister to panic on duplicate, but it did not")
		}
	}()
	MustRegister(&AppConfig{Name: "t_must_panic"})
}

func TestRegisterDefaultsLabelAndVerboseName(t *testing.T) {
	Register(&AppConfig{Name: "t_defaults"})
	cfg, _ := GetAppConfig("t_defaults")
	if cfg.Label != "t_defaults" {
		t.Errorf("expected Label to default to Name, got: %s", cfg.Label)
	}
	if cfg.VerboseName != "t_defaults" {
		t.Errorf("expected VerboseName to default to Name, got: %s", cfg.VerboseName)
	}
}

func TestRegisterEmptyNameReturnsError(t *testing.T) {
	err := Register(&AppConfig{Name: ""})
	if err == nil {
		t.Fatal("expected error for empty app name, got nil")
	}
}

func TestGetAppConfig(t *testing.T) {
	Register(&AppConfig{Name: "t_get", Label: "getlabel", VerboseName: "Get App"})
	cfg, err := GetAppConfig("t_get")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.Name != "t_get" {
		t.Errorf("expected Name 't_get', got: %s", cfg.Name)
	}
	if cfg.Label != "getlabel" {
		t.Errorf("expected Label 'getlabel', got: %s", cfg.Label)
	}
}

func TestGetAppConfigNotFound(t *testing.T) {
	_, err := GetAppConfig("t_nonexistent_xyz")
	if err == nil {
		t.Fatal("expected error for unknown app, got nil")
	}
	if !strings.Contains(err.Error(), "no app with name") {
		t.Fatalf("expected 'no app with name' in error, got: %v", err)
	}
}

func TestIsInstalled(t *testing.T) {
	Register(&AppConfig{Name: "t_installed"})
	if !IsInstalled("t_installed") {
		t.Error("expected IsInstalled to return true for registered app")
	}
	if IsInstalled("t_not_installed_xyz") {
		t.Error("expected IsInstalled to return false for unknown app")
	}
}

func TestGetAppConfigs(t *testing.T) {
	Register(&AppConfig{Name: "t_configs_a"})
	Register(&AppConfig{Name: "t_configs_b"})

	configs := GetAppConfigs()
	names := make(map[string]bool)
	for _, c := range configs {
		names[c.Name] = true
	}
	if !names["t_configs_a"] {
		t.Error("expected t_configs_a in GetAppConfigs()")
	}
	if !names["t_configs_b"] {
		t.Error("expected t_configs_b in GetAppConfigs()")
	}
}

func TestGetAppConfigsPreservesOrder(t *testing.T) {
	Register(&AppConfig{Name: "t_order_1"})
	Register(&AppConfig{Name: "t_order_2"})
	Register(&AppConfig{Name: "t_order_3"})

	configs := GetAppConfigs()
	// Find positions of our three apps in the ordered list
	pos := map[string]int{}
	for i, c := range configs {
		pos[c.Name] = i
	}
	if pos["t_order_1"] >= pos["t_order_2"] || pos["t_order_2"] >= pos["t_order_3"] {
		t.Error("expected registration order to be preserved")
	}
}

func TestAppNames(t *testing.T) {
	Register(&AppConfig{Name: "t_names_x"})
	Register(&AppConfig{Name: "t_names_y"})

	names := AppNames()
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["t_names_x"] || !found["t_names_y"] {
		t.Error("expected both apps in AppNames()")
	}
}

func TestReadyFnCalled(t *testing.T) {
	called := false
	cfg := &AppConfig{Name: "t_ready_fn"}
	cfg.SetReady(func() { called = true })
	Register(cfg)

	cfg.Ready()
	if !called {
		t.Error("expected Ready() to call the readyFn")
	}
}

func TestSetupCallsReadyOnInstalledApps(t *testing.T) {
	calledA := false
	calledB := false

	cfgA := &AppConfig{Name: "t_setup_a"}
	cfgA.SetReady(func() { calledA = true })
	cfgB := &AppConfig{Name: "t_setup_b"}
	cfgB.SetReady(func() { calledB = true })

	Register(cfgA)
	Register(cfgB)

	// Reset ready flag so Setup() runs (it's idempotent after first call)
	registry.mu.Lock()
	registry.ready = false
	registry.mu.Unlock()

	err := Setup([]string{"t_setup_a", "t_setup_b"})
	if err != nil {
		t.Fatalf("expected no error from Setup(), got: %v", err)
	}
	if !calledA {
		t.Error("expected Ready() to be called for t_setup_a")
	}
	if !calledB {
		t.Error("expected Ready() to be called for t_setup_b")
	}
}

func TestSetupUnregisteredAppReturnsError(t *testing.T) {
	registry.mu.Lock()
	registry.ready = false
	registry.mu.Unlock()

	err := Setup([]string{"t_never_registered_xyz"})
	if err == nil {
		t.Fatal("expected error for unregistered app in InstalledApps, got nil")
	}
	if !strings.Contains(err.Error(), "never registered") {
		t.Fatalf("expected 'never registered' in error, got: %v", err)
	}
}
