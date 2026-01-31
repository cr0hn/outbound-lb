package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadFromFile_AllFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "full_config.yml")

	configContent := `
ips:
  - 10.0.0.1
  - 10.0.0.2
  - 10.0.0.3
port: 8888
metrics_port: 9999
auth: "admin:secret123"
timeout: 45s
idle_timeout: 90s
max_conns_per_ip: 50
max_conns_total: 500
history_window: 10m
history_size: 200
log_level: debug
log_format: text
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error: %v", err)
	}

	// Verify all fields
	if len(cfg.IPs) != 3 {
		t.Errorf("expected 3 IPs, got %d", len(cfg.IPs))
	}
	if cfg.Port != 8888 {
		t.Errorf("expected port 8888, got %d", cfg.Port)
	}
	if cfg.MetricsPort != 9999 {
		t.Errorf("expected metrics port 9999, got %d", cfg.MetricsPort)
	}
	if cfg.Auth != "admin:secret123" {
		t.Errorf("expected auth 'admin:secret123', got %s", cfg.Auth)
	}
	if cfg.Timeout != 45*time.Second {
		t.Errorf("expected timeout 45s, got %v", cfg.Timeout)
	}
	if cfg.IdleTimeout != 90*time.Second {
		t.Errorf("expected idle timeout 90s, got %v", cfg.IdleTimeout)
	}
	if cfg.MaxConnsPerIP != 50 {
		t.Errorf("expected max conns per IP 50, got %d", cfg.MaxConnsPerIP)
	}
	if cfg.MaxConnsTotal != 500 {
		t.Errorf("expected max conns total 500, got %d", cfg.MaxConnsTotal)
	}
	if cfg.HistoryWindow != 10*time.Minute {
		t.Errorf("expected history window 10m, got %v", cfg.HistoryWindow)
	}
	if cfg.HistorySize != 200 {
		t.Errorf("expected history size 200, got %d", cfg.HistorySize)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected log level 'debug', got %s", cfg.LogLevel)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("expected log format 'text', got %s", cfg.LogFormat)
	}
}

func TestLoadFromFile_MinimalValid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "minimal.yml")

	configContent := `
ips:
  - 127.0.0.1
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error: %v", err)
	}

	// Defaults should be applied
	if cfg.Port != 3128 {
		t.Errorf("expected default port 3128, got %d", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default log level 'info', got %s", cfg.LogLevel)
	}
}

func TestLoadFromFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "empty.yml")

	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error: %v", err)
	}

	// Should get defaults
	if cfg.Port != 3128 {
		t.Errorf("expected default port, got %d", cfg.Port)
	}
}

func TestConfig_Validate_AllLogLevels(t *testing.T) {
	validLevels := []string{"debug", "info", "warn", "error"}

	for _, level := range validLevels {
		cfg := DefaultConfig()
		cfg.IPs = []string{"127.0.0.1"}
		cfg.LogLevel = level

		err := cfg.Validate()
		if err != nil {
			t.Errorf("log level '%s' should be valid, got error: %v", level, err)
		}
	}
}

func TestConfig_Validate_AllLogFormats(t *testing.T) {
	validFormats := []string{"json", "text"}

	for _, format := range validFormats {
		cfg := DefaultConfig()
		cfg.IPs = []string{"127.0.0.1"}
		cfg.LogFormat = format

		err := cfg.Validate()
		if err != nil {
			t.Errorf("log format '%s' should be valid, got error: %v", format, err)
		}
	}
}

func TestConfig_Validate_MultipleIPs(t *testing.T) {
	cfg := DefaultConfig()
	cfg.IPs = []string{"10.0.0.1", "10.0.0.2", "10.0.0.3", "192.168.1.1"}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("multiple valid IPs should pass validation: %v", err)
	}
}

func TestConfig_Validate_IPv6(t *testing.T) {
	cfg := DefaultConfig()
	cfg.IPs = []string{"::1", "2001:db8::1"}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("IPv6 addresses should be valid: %v", err)
	}
}

func TestConfig_Validate_MixedIPVersions(t *testing.T) {
	cfg := DefaultConfig()
	cfg.IPs = []string{"192.168.1.1", "::1"}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("mixed IP versions should be valid: %v", err)
	}
}

func TestConfig_Validate_MetricsPorts(t *testing.T) {
	cfg := DefaultConfig()
	cfg.IPs = []string{"127.0.0.1"}
	cfg.MetricsPort = 0

	err := cfg.Validate()
	if err == nil {
		t.Error("metrics port 0 should be invalid")
	}

	cfg.MetricsPort = 70000
	err = cfg.Validate()
	if err == nil {
		t.Error("metrics port > 65535 should be invalid")
	}
}
