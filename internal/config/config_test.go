package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeYAML(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != ":3000" {
		t.Fatalf("listen = %q", cfg.Listen)
	}
	if cfg.Database.Path != "./data/kui.db" {
		t.Fatalf("db = %q", cfg.Database.Path)
	}
	if cfg.Kiko.URL != "http://127.0.0.1:8080" {
		t.Fatalf("kiko url = %q", cfg.Kiko.URL)
	}
	if cfg.DefaultLocale != "en" {
		t.Fatalf("locale = %q", cfg.DefaultLocale)
	}
	if cfg.Admin.Email != "admin@localhost" {
		t.Fatalf("admin email = %q", cfg.Admin.Email)
	}
	if cfg.Log == nil {
		t.Fatal("logger should be initialized")
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	yml := filepath.Join(dir, "kui.yml")
	writeYAML(t, yml, `
listen: ":9999"
database:
  path: /tmp/test.db
kiko:
  url: http://kiko:8080
  api_key: test-key
`)
	cfg, err := Load(yml)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != ":9999" {
		t.Fatalf("listen = %q", cfg.Listen)
	}
	if cfg.Database.Path != "/tmp/test.db" {
		t.Fatalf("db = %q", cfg.Database.Path)
	}
	if cfg.Kiko.APIKey != "test-key" {
		t.Fatalf("api key = %q", cfg.Kiko.APIKey)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	t.Setenv("KUI_LISTEN", ":4000")
	t.Setenv("KIKO_API_KEY", "env-key")
	t.Setenv("KUI_DATABASE_PATH", "/env/db.sqlite")

	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != ":4000" {
		t.Fatalf("env listen = %q", cfg.Listen)
	}
	if cfg.Kiko.APIKey != "env-key" {
		t.Fatalf("env api key = %q", cfg.Kiko.APIKey)
	}
	if cfg.Database.Path != "/env/db.sqlite" {
		t.Fatalf("env db = %q", cfg.Database.Path)
	}
}

func TestLoadEnvSession(t *testing.T) {
	t.Setenv("KUI_SESSION_COOKIE", "my_cookie")
	t.Setenv("KUI_SESSION_TTL_HOURS", "24")
	t.Setenv("KUI_SESSION_SHORT_TTL_HOURS", "2")
	t.Setenv("KUI_SESSION_SECURE", "true")

	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Session.CookieName != "my_cookie" {
		t.Fatalf("cookie = %q", cfg.Session.CookieName)
	}
	if cfg.Session.TTLHours != 24 {
		t.Fatalf("ttl = %d", cfg.Session.TTLHours)
	}
	if cfg.Session.ShortTTLHours != 2 {
		t.Fatalf("short ttl = %d", cfg.Session.ShortTTLHours)
	}
	if !cfg.Session.Secure {
		t.Fatal("session secure should be true")
	}
}

func TestLoadEnvAdmin(t *testing.T) {
	t.Setenv("KUI_ADMIN_EMAIL", "root@test")
	t.Setenv("KUI_ADMIN_PASSWORD", "secret")

	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Admin.Email != "root@test" {
		t.Fatalf("admin email = %q", cfg.Admin.Email)
	}
	if cfg.Admin.Password != "secret" {
		t.Fatalf("admin password = %q", cfg.Admin.Password)
	}
}

func TestLoadEnvLocale(t *testing.T) {
	t.Setenv("KUI_DEFAULT_LOCALE", "de")
	t.Setenv("KUI_ENABLED_LOCALES", "en,de,es")

	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultLocale != "de" {
		t.Fatalf("default locale = %q", cfg.DefaultLocale)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/kui.yml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadLogLevel(t *testing.T) {
	t.Setenv("KUI_LOG_LEVEL", "debug")
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Log == nil {
		t.Fatal("logger should be non-nil")
	}
}
