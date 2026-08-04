package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DB_DRIVER", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.Driver != "postgres" {
		t.Fatalf("Database.Driver = %q, want postgres", cfg.Database.Driver)
	}
	if cfg.JWT.TTL != 24*time.Hour {
		t.Fatalf("JWT.TTL = %s, want 24h", cfg.JWT.TTL)
	}
}

func TestLoadRejectsInvalidDriver(t *testing.T) {
	t.Setenv("DB_DRIVER", "sqlite")
	t.Setenv("JWT_SECRET", "development-only-secret-change-before-release")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid driver error")
	}
}

func TestLoadRequiresProductionSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want JWT secret error")
	}
}
