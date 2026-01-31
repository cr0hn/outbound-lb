package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Port != 3128 {
		t.Errorf("expected default port 3128, got %d", cfg.Port)
	}
	if cfg.MetricsPort != 9090 {
		t.Errorf("expected default metrics port 9090, got %d", cfg.MetricsPort)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", cfg.Timeout)
	}
	if cfg.MaxConnsPerIP != 100 {
		t.Errorf("expected default max conns per IP 100, got %d", cfg.MaxConnsPerIP)
	}
	if cfg.MaxConnsTotal != 1000 {
		t.Errorf("expected default max conns total 1000, got %d", cfg.MaxConnsTotal)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default log level 'info', got %s", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("expected default log format 'json', got %s", cfg.LogFormat)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid config",
			modify:  func(c *Config) { c.IPs = []string{"192.168.1.1"} },
			wantErr: false,
		},
		{
			name:    "no IPs",
			modify:  func(c *Config) {},
			wantErr: true,
		},
		{
			name:    "invalid IP",
			modify:  func(c *Config) { c.IPs = []string{"invalid"} },
			wantErr: true,
		},
		{
			name:    "invalid port - zero",
			modify:  func(c *Config) { c.IPs = []string{"192.168.1.1"}; c.Port = 0 },
			wantErr: true,
		},
		{
			name:    "invalid port - too high",
			modify:  func(c *Config) { c.IPs = []string{"192.168.1.1"}; c.Port = 70000 },
			wantErr: true,
		},
		{
			name:    "same port for proxy and metrics",
			modify:  func(c *Config) { c.IPs = []string{"192.168.1.1"}; c.Port = 3128; c.MetricsPort = 3128 },
			wantErr: true,
		},
		{
			name:    "invalid auth format",
			modify:  func(c *Config) { c.IPs = []string{"192.168.1.1"}; c.Auth = "nocolon" },
			wantErr: true,
		},
		{
			name:    "valid auth",
			modify:  func(c *Config) { c.IPs = []string{"192.168.1.1"}; c.Auth = "user:pass" },
			wantErr: false,
		},
		{
			name:    "invalid timeout",
			modify:  func(c *Config) { c.IPs = []string{"192.168.1.1"}; c.Timeout = 0 },
			wantErr: true,
		},
		{
			name:    "invalid idle timeout",
			modify:  func(c *Config) { c.IPs = []string{"192.168.1.1"}; c.IdleTimeout = 0 },
			wantErr: true,
		},
		{
			name:    "invalid max conns per IP",
			modify:  func(c *Config) { c.IPs = []string{"192.168.1.1"}; c.MaxConnsPerIP = 0 },
			wantErr: true,
		},
		{
			name:    "invalid max conns total",
			modify:  func(c *Config) { c.IPs = []string{"192.168.1.1"}; c.MaxConnsTotal = 0 },
			wantErr: true,
		},
		{
			name:    "invalid history window",
			modify:  func(c *Config) { c.IPs = []string{"192.168.1.1"}; c.HistoryWindow = 0 },
			wantErr: true,
		},
		{
			name:    "invalid history size",
			modify:  func(c *Config) { c.IPs = []string{"192.168.1.1"}; c.HistorySize = 0 },
			wantErr: true,
		},
		{
			name:    "invalid log level",
			modify:  func(c *Config) { c.IPs = []string{"192.168.1.1"}; c.LogLevel = "invalid" },
			wantErr: true,
		},
		{
			name:    "invalid log format",
			modify:  func(c *Config) { c.IPs = []string{"192.168.1.1"}; c.LogFormat = "invalid" },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigGetAuthCredentials(t *testing.T) {
	tests := []struct {
		name     string
		auth     string
		wantUser string
		wantPass string
		wantOk   bool
	}{
		{"no auth", "", "", "", false},
		{"valid auth", "user:pass", "user", "pass", true},
		{"password with colon", "user:pass:word", "user", "pass:word", true},
		{"invalid format", "nocolon", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Auth: tt.auth}
			user, pass, ok := cfg.GetAuthCredentials()
			if user != tt.wantUser || pass != tt.wantPass || ok != tt.wantOk {
				t.Errorf("GetAuthCredentials() = (%q, %q, %v), want (%q, %q, %v)",
					user, pass, ok, tt.wantUser, tt.wantPass, tt.wantOk)
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	configContent := `
ips:
  - 192.168.1.1
  - 192.168.1.2
port: 8080
metrics_port: 9091
auth: "testuser:testpass"
timeout: 60s
log_level: debug
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error: %v", err)
	}

	if len(cfg.IPs) != 2 {
		t.Errorf("expected 2 IPs, got %d", len(cfg.IPs))
	}
	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if cfg.Auth != "testuser:testpass" {
		t.Errorf("expected auth 'testuser:testpass', got %s", cfg.Auth)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected log level 'debug', got %s", cfg.LogLevel)
	}
}

func TestLoadFromFile_NotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/config.yml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadFromFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yml")

	if err := os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadFromFile(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}
