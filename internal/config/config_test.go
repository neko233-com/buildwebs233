package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesDefaultsAndOverrides(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "server.yaml")
	content := []byte(`
server:
  port: 7001
auth:
  username: "admin"
storage:
  data_dir: "custom-data"
`)
	if err := os.WriteFile(cfgPath, content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Server.Port != 7001 {
		t.Fatalf("expected port 7001, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Fatalf("expected default host, got %q", cfg.Server.Host)
	}
	if cfg.Auth.Username != "admin" {
		t.Fatalf("expected username override, got %q", cfg.Auth.Username)
	}
	if cfg.Auth.Password != "root" {
		t.Fatalf("expected default password, got %q", cfg.Auth.Password)
	}
	if cfg.Storage.DataDir != "custom-data" {
		t.Fatalf("expected custom data dir, got %q", cfg.Storage.DataDir)
	}
	if len(cfg.HotReload.WatchPath) == 0 {
		t.Fatal("expected default watch paths")
	}
}
