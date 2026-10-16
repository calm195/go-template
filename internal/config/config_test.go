package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvironmentOverridesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  address: ':9000'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_SERVER_ADDRESS", ":9100")
	t.Setenv("APP_REDIS_ENABLED", "true")
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Server.Address != ":9100" || !c.Redis.Enabled {
		t.Fatalf("environment not applied: %+v", c.Server)
	}
}

func TestLoadRejectsInvalidDriver(t *testing.T) {
	t.Setenv("APP_DATABASE_DRIVER", "sqlite")
	if _, err := Load(""); err == nil {
		t.Fatal("expected invalid driver error")
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("expected missing file error")
	}
}
