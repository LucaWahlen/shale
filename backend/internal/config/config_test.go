package config

import (
	"strings"
	"testing"
)

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	t.Setenv(key, value)
}

func TestLoadRequiresAdminPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "unset", password: "", wantErr: true},
		{name: "empty", password: "", wantErr: true},
		{name: "set", password: "secret", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.password != "" {
				setEnv(t, "SHALE_ADMIN_PASSWORD", tt.password)
			} else {
				t.Setenv("SHALE_ADMIN_PASSWORD", "")
			}
			cfg, err := Load()
			if tt.wantErr != (err != nil) {
				t.Fatalf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && cfg.AdminPassword != tt.password {
				t.Fatalf("AdminPassword = %q, want %q", cfg.AdminPassword, tt.password)
			}
		})
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("SHALE_ADMIN_PASSWORD", "pw")
	t.Setenv("SHALE_DB_PATH", "")
	t.Setenv("SHALE_ADDR", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DBPath != "/data/shale.db" {
		t.Errorf("DBPath = %q, want /data/shale.db", cfg.DBPath)
	}
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080", cfg.Addr)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("SHALE_ADMIN_PASSWORD", "p")
	t.Setenv("SHALE_DB_PATH", "data/shale.db")
	t.Setenv("SHALE_ADDR", ":9999")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DBPath != "data/shale.db" || cfg.Addr != ":9999" {
		t.Errorf("unexpected config: %+v", cfg)
	}
	if strings.Contains(cfg.DBPath, "/data") {
		t.Errorf("dev override ignored")
	}
}
