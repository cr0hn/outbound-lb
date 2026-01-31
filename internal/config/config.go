// Package config handles configuration parsing from CLI flags and YAML files.
package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the proxy.
type Config struct {
	// IPs is the list of outbound IPs to use for load balancing.
	IPs []string `yaml:"ips"`
	// Port is the proxy listening port.
	Port int `yaml:"port"`
	// MetricsPort is the metrics server port.
	MetricsPort int `yaml:"metrics_port"`
	// Auth is the optional basic auth in "user:pass" format.
	Auth string `yaml:"auth"`
	// Timeout is the connection timeout.
	Timeout time.Duration `yaml:"timeout"`
	// IdleTimeout is the idle connection timeout.
	IdleTimeout time.Duration `yaml:"idle_timeout"`
	// MaxConnsPerIP is the maximum concurrent connections per outbound IP.
	MaxConnsPerIP int `yaml:"max_conns_per_ip"`
	// MaxConnsTotal is the maximum total concurrent connections.
	MaxConnsTotal int `yaml:"max_conns_total"`
	// HistoryWindow is the time window for LRU history.
	HistoryWindow time.Duration `yaml:"history_window"`
	// HistorySize is the max entries per host in history.
	HistorySize int `yaml:"history_size"`
	// HistoryMaxTotalEntries is the maximum total entries across all hosts.
	HistoryMaxTotalEntries int `yaml:"history_max_total_entries"`
	// LogLevel is the logging level (debug, info, warn, error).
	LogLevel string `yaml:"log_level"`
	// LogFormat is the log format (json, text).
	LogFormat string `yaml:"log_format"`
	// ConfigFile is the optional config file path.
	ConfigFile string `yaml:"-"`

	// Transport tuning
	// TCPKeepAlive is the TCP keep-alive interval.
	TCPKeepAlive time.Duration `yaml:"tcp_keepalive"`
	// IdleConnTimeout is the timeout for idle HTTP connections.
	IdleConnTimeout time.Duration `yaml:"idle_conn_timeout"`
	// TLSHandshakeTimeout is the timeout for TLS handshakes.
	TLSHandshakeTimeout time.Duration `yaml:"tls_handshake_timeout"`
	// ExpectContinueTimeout is the timeout for 100-continue responses.
	ExpectContinueTimeout time.Duration `yaml:"expect_continue_timeout"`

	// Circuit Breaker configuration
	// CircuitBreakerEnabled enables the circuit breaker per IP.
	CircuitBreakerEnabled bool `yaml:"circuit_breaker_enabled"`
	// CBFailureThreshold is the number of failures before opening the circuit.
	CBFailureThreshold int `yaml:"cb_failure_threshold"`
	// CBSuccessThreshold is the number of successes in half-open to close the circuit.
	CBSuccessThreshold int `yaml:"cb_success_threshold"`
	// CBTimeout is how long the circuit stays open before transitioning to half-open.
	CBTimeout time.Duration `yaml:"cb_timeout"`

	// Health Check configuration
	// HealthCheckEnabled enables active health checks for outbound IPs.
	HealthCheckEnabled bool `yaml:"health_check_enabled"`
	// HealthCheckType is the type of health check: "tcp" or "http".
	HealthCheckType string `yaml:"health_check_type"`
	// HealthCheckInterval is the interval between health checks.
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	// HealthCheckTimeout is the timeout for each health check.
	HealthCheckTimeout time.Duration `yaml:"health_check_timeout"`
	// HealthCheckTarget is the target for health checks (host:port for TCP, URL for HTTP).
	HealthCheckTarget string `yaml:"health_check_target"`
	// HealthCheckFailureThreshold is the number of failures before marking an IP unhealthy.
	HealthCheckFailureThreshold int `yaml:"health_check_failure_threshold"`
	// HealthCheckSuccessThreshold is the number of successes before marking an IP healthy.
	HealthCheckSuccessThreshold int `yaml:"health_check_success_threshold"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Port:                   3128,
		MetricsPort:            9090,
		Timeout:                30 * time.Second,
		IdleTimeout:            60 * time.Second,
		MaxConnsPerIP:          100,
		MaxConnsTotal:          1000,
		HistoryWindow:          5 * time.Minute,
		HistorySize:            100,
		HistoryMaxTotalEntries: 100000,
		LogLevel:               "info",
		LogFormat:              "json",
		// Transport defaults
		TCPKeepAlive:          30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// Circuit breaker defaults
		CircuitBreakerEnabled: false,
		CBFailureThreshold:    5,
		CBSuccessThreshold:    2,
		CBTimeout:             30 * time.Second,
		// Health check defaults
		HealthCheckEnabled:          false,
		HealthCheckType:             "tcp",
		HealthCheckInterval:         10 * time.Second,
		HealthCheckTimeout:          5 * time.Second,
		HealthCheckTarget:           "1.1.1.1:443",
		HealthCheckFailureThreshold: 3,
		HealthCheckSuccessThreshold: 2,
	}
}

// ParseFlags parses command line flags and returns a Config.
func ParseFlags() (*Config, error) {
	cfg := DefaultConfig()

	pflag.StringSliceVar(&cfg.IPs, "ips", nil, "Comma-separated list of outbound IPs")
	pflag.IntVar(&cfg.Port, "port", cfg.Port, "Proxy listening port")
	pflag.IntVar(&cfg.MetricsPort, "metrics-port", cfg.MetricsPort, "Metrics server port")
	pflag.StringVar(&cfg.Auth, "auth", "", "Basic auth credentials (user:pass)")
	pflag.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "Connection timeout")
	pflag.DurationVar(&cfg.IdleTimeout, "idle-timeout", cfg.IdleTimeout, "Idle connection timeout")
	pflag.IntVar(&cfg.MaxConnsPerIP, "max-conns-per-ip", cfg.MaxConnsPerIP, "Max connections per outbound IP")
	pflag.IntVar(&cfg.MaxConnsTotal, "max-conns-total", cfg.MaxConnsTotal, "Max total connections")
	pflag.DurationVar(&cfg.HistoryWindow, "history-window", cfg.HistoryWindow, "LRU history time window")
	pflag.IntVar(&cfg.HistorySize, "history-size", cfg.HistorySize, "Max history entries per host")
	pflag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "Log level (debug, info, warn, error)")
	pflag.StringVar(&cfg.LogFormat, "log-format", cfg.LogFormat, "Log format (json, text)")
	pflag.StringVar(&cfg.ConfigFile, "config", "", "Config file path (YAML)")

	// Transport tuning flags
	pflag.DurationVar(&cfg.TCPKeepAlive, "tcp-keepalive", cfg.TCPKeepAlive, "TCP keep-alive interval")
	pflag.DurationVar(&cfg.IdleConnTimeout, "idle-conn-timeout", cfg.IdleConnTimeout, "Idle HTTP connection timeout")
	pflag.DurationVar(&cfg.TLSHandshakeTimeout, "tls-handshake-timeout", cfg.TLSHandshakeTimeout, "TLS handshake timeout")
	pflag.DurationVar(&cfg.ExpectContinueTimeout, "expect-continue-timeout", cfg.ExpectContinueTimeout, "Expect-continue timeout")
	pflag.IntVar(&cfg.HistoryMaxTotalEntries, "history-max-total-entries", cfg.HistoryMaxTotalEntries, "Max total history entries")

	// Circuit breaker flags
	pflag.BoolVar(&cfg.CircuitBreakerEnabled, "circuit-breaker-enabled", cfg.CircuitBreakerEnabled, "Enable circuit breaker")
	pflag.IntVar(&cfg.CBFailureThreshold, "cb-failure-threshold", cfg.CBFailureThreshold, "Circuit breaker failure threshold")
	pflag.IntVar(&cfg.CBSuccessThreshold, "cb-success-threshold", cfg.CBSuccessThreshold, "Circuit breaker success threshold")
	pflag.DurationVar(&cfg.CBTimeout, "cb-timeout", cfg.CBTimeout, "Circuit breaker timeout")

	// Health check flags
	pflag.BoolVar(&cfg.HealthCheckEnabled, "health-check-enabled", cfg.HealthCheckEnabled, "Enable active health checks")
	pflag.StringVar(&cfg.HealthCheckType, "health-check-type", cfg.HealthCheckType, "Health check type: tcp or http")
	pflag.DurationVar(&cfg.HealthCheckInterval, "health-check-interval", cfg.HealthCheckInterval, "Health check interval")
	pflag.DurationVar(&cfg.HealthCheckTimeout, "health-check-timeout", cfg.HealthCheckTimeout, "Health check timeout")
	pflag.StringVar(&cfg.HealthCheckTarget, "health-check-target", cfg.HealthCheckTarget, "Health check target (host:port for tcp, URL for http)")
	pflag.IntVar(&cfg.HealthCheckFailureThreshold, "health-check-failure-threshold", cfg.HealthCheckFailureThreshold, "Failures before marking IP unhealthy")
	pflag.IntVar(&cfg.HealthCheckSuccessThreshold, "health-check-success-threshold", cfg.HealthCheckSuccessThreshold, "Successes before marking IP healthy")

	pflag.Parse()

	// Load from environment variables (env vars take precedence over defaults, but CLI flags take precedence over env vars)
	loadFromEnv(cfg)

	// If config file specified, load it first, then override with flags
	if cfg.ConfigFile != "" {
		fileCfg, err := LoadFromFile(cfg.ConfigFile)
		if err != nil {
			return nil, fmt.Errorf("loading config file: %w", err)
		}
		cfg = mergeConfigs(fileCfg, cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return cfg, nil
}

// LoadFromFile loads configuration from a YAML file.
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return cfg, nil
}

// mergeConfigs merges file config with CLI config. CLI flags take precedence.
func mergeConfigs(file, cli *Config) *Config {
	result := *file

	// Check if flag was explicitly set
	pflag.Visit(func(f *pflag.Flag) {
		switch f.Name {
		case "ips":
			result.IPs = cli.IPs
		case "port":
			result.Port = cli.Port
		case "metrics-port":
			result.MetricsPort = cli.MetricsPort
		case "auth":
			result.Auth = cli.Auth
		case "timeout":
			result.Timeout = cli.Timeout
		case "idle-timeout":
			result.IdleTimeout = cli.IdleTimeout
		case "max-conns-per-ip":
			result.MaxConnsPerIP = cli.MaxConnsPerIP
		case "max-conns-total":
			result.MaxConnsTotal = cli.MaxConnsTotal
		case "history-window":
			result.HistoryWindow = cli.HistoryWindow
		case "history-size":
			result.HistorySize = cli.HistorySize
		case "log-level":
			result.LogLevel = cli.LogLevel
		case "log-format":
			result.LogFormat = cli.LogFormat
		case "health-check-enabled":
			result.HealthCheckEnabled = cli.HealthCheckEnabled
		case "health-check-type":
			result.HealthCheckType = cli.HealthCheckType
		case "health-check-interval":
			result.HealthCheckInterval = cli.HealthCheckInterval
		case "health-check-timeout":
			result.HealthCheckTimeout = cli.HealthCheckTimeout
		case "health-check-target":
			result.HealthCheckTarget = cli.HealthCheckTarget
		case "health-check-failure-threshold":
			result.HealthCheckFailureThreshold = cli.HealthCheckFailureThreshold
		case "health-check-success-threshold":
			result.HealthCheckSuccessThreshold = cli.HealthCheckSuccessThreshold
		case "tcp-keepalive":
			result.TCPKeepAlive = cli.TCPKeepAlive
		case "idle-conn-timeout":
			result.IdleConnTimeout = cli.IdleConnTimeout
		case "tls-handshake-timeout":
			result.TLSHandshakeTimeout = cli.TLSHandshakeTimeout
		case "expect-continue-timeout":
			result.ExpectContinueTimeout = cli.ExpectContinueTimeout
		case "history-max-total-entries":
			result.HistoryMaxTotalEntries = cli.HistoryMaxTotalEntries
		case "circuit-breaker-enabled":
			result.CircuitBreakerEnabled = cli.CircuitBreakerEnabled
		case "cb-failure-threshold":
			result.CBFailureThreshold = cli.CBFailureThreshold
		case "cb-success-threshold":
			result.CBSuccessThreshold = cli.CBSuccessThreshold
		case "cb-timeout":
			result.CBTimeout = cli.CBTimeout
		}
	})

	return &result
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if len(c.IPs) == 0 {
		return fmt.Errorf("at least one outbound IP is required (--ips)")
	}

	for _, ip := range c.IPs {
		if net.ParseIP(ip) == nil {
			return fmt.Errorf("invalid IP address: %s", ip)
		}
	}

	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if c.MetricsPort < 1 || c.MetricsPort > 65535 {
		return fmt.Errorf("invalid metrics port: %d", c.MetricsPort)
	}

	if c.Port == c.MetricsPort {
		return fmt.Errorf("proxy port and metrics port must be different")
	}

	if c.Auth != "" && !strings.Contains(c.Auth, ":") {
		return fmt.Errorf("auth must be in 'user:pass' format")
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if c.IdleTimeout <= 0 {
		return fmt.Errorf("idle-timeout must be positive")
	}

	if c.MaxConnsPerIP < 1 {
		return fmt.Errorf("max-conns-per-ip must be at least 1")
	}

	if c.MaxConnsTotal < 1 {
		return fmt.Errorf("max-conns-total must be at least 1")
	}

	if c.HistoryWindow <= 0 {
		return fmt.Errorf("history-window must be positive")
	}

	if c.HistorySize < 1 {
		return fmt.Errorf("history-size must be at least 1")
	}

	validLevels := map[string]bool{"trace": true, "debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s (must be trace, debug, info, warn, or error)", c.LogLevel)
	}

	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[c.LogFormat] {
		return fmt.Errorf("invalid log format: %s (must be json or text)", c.LogFormat)
	}

	return nil
}

// GetAuthCredentials returns username and password if auth is configured.
func (c *Config) GetAuthCredentials() (username, password string, ok bool) {
	if c.Auth == "" {
		return "", "", false
	}
	parts := strings.SplitN(c.Auth, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// loadFromEnv loads configuration from environment variables with OUTBOUND_LB_ prefix.
// Environment variables take precedence over defaults but CLI flags take precedence over env vars.
func loadFromEnv(cfg *Config) {
	// Helper functions for parsing
	getEnvString := func(key string) (string, bool) {
		v := os.Getenv("OUTBOUND_LB_" + key)
		return v, v != ""
	}

	getEnvInt := func(key string) (int, bool) {
		if v, ok := getEnvString(key); ok {
			if i, err := strconv.Atoi(v); err == nil {
				return i, true
			}
		}
		return 0, false
	}

	getEnvBool := func(key string) (bool, bool) {
		if v, ok := getEnvString(key); ok {
			if b, err := strconv.ParseBool(v); err == nil {
				return b, true
			}
		}
		return false, false
	}

	getEnvDuration := func(key string) (time.Duration, bool) {
		if v, ok := getEnvString(key); ok {
			if d, err := time.ParseDuration(v); err == nil {
				return d, true
			}
		}
		return 0, false
	}

	// Only apply env vars if CLI flag was not explicitly set
	applyIfNotSet := func(flagName string, apply func()) {
		flagSet := false
		pflag.Visit(func(f *pflag.Flag) {
			if f.Name == flagName {
				flagSet = true
			}
		})
		if !flagSet {
			apply()
		}
	}

	// Server configuration
	if v, ok := getEnvString("IPS"); ok {
		applyIfNotSet("ips", func() {
			cfg.IPs = strings.Split(v, ",")
			// Trim whitespace from each IP
			for i, ip := range cfg.IPs {
				cfg.IPs[i] = strings.TrimSpace(ip)
			}
		})
	}

	if v, ok := getEnvInt("PORT"); ok {
		applyIfNotSet("port", func() { cfg.Port = v })
	}

	if v, ok := getEnvInt("METRICS_PORT"); ok {
		applyIfNotSet("metrics-port", func() { cfg.MetricsPort = v })
	}

	if v, ok := getEnvString("AUTH"); ok {
		applyIfNotSet("auth", func() { cfg.Auth = v })
	}

	// Timeouts
	if v, ok := getEnvDuration("TIMEOUT"); ok {
		applyIfNotSet("timeout", func() { cfg.Timeout = v })
	}

	if v, ok := getEnvDuration("IDLE_TIMEOUT"); ok {
		applyIfNotSet("idle-timeout", func() { cfg.IdleTimeout = v })
	}

	// Connection limits
	if v, ok := getEnvInt("MAX_CONNS_PER_IP"); ok {
		applyIfNotSet("max-conns-per-ip", func() { cfg.MaxConnsPerIP = v })
	}

	if v, ok := getEnvInt("MAX_CONNS_TOTAL"); ok {
		applyIfNotSet("max-conns-total", func() { cfg.MaxConnsTotal = v })
	}

	// Load balancer settings
	if v, ok := getEnvDuration("HISTORY_WINDOW"); ok {
		applyIfNotSet("history-window", func() { cfg.HistoryWindow = v })
	}

	if v, ok := getEnvInt("HISTORY_SIZE"); ok {
		applyIfNotSet("history-size", func() { cfg.HistorySize = v })
	}

	if v, ok := getEnvInt("HISTORY_MAX_TOTAL_ENTRIES"); ok {
		applyIfNotSet("history-max-total-entries", func() { cfg.HistoryMaxTotalEntries = v })
	}

	// Logging
	if v, ok := getEnvString("LOG_LEVEL"); ok {
		applyIfNotSet("log-level", func() { cfg.LogLevel = v })
	}

	if v, ok := getEnvString("LOG_FORMAT"); ok {
		applyIfNotSet("log-format", func() { cfg.LogFormat = v })
	}

	// Transport tuning
	if v, ok := getEnvDuration("TCP_KEEPALIVE"); ok {
		applyIfNotSet("tcp-keepalive", func() { cfg.TCPKeepAlive = v })
	}

	if v, ok := getEnvDuration("IDLE_CONN_TIMEOUT"); ok {
		applyIfNotSet("idle-conn-timeout", func() { cfg.IdleConnTimeout = v })
	}

	if v, ok := getEnvDuration("TLS_HANDSHAKE_TIMEOUT"); ok {
		applyIfNotSet("tls-handshake-timeout", func() { cfg.TLSHandshakeTimeout = v })
	}

	if v, ok := getEnvDuration("EXPECT_CONTINUE_TIMEOUT"); ok {
		applyIfNotSet("expect-continue-timeout", func() { cfg.ExpectContinueTimeout = v })
	}

	// Circuit breaker
	if v, ok := getEnvBool("CIRCUIT_BREAKER_ENABLED"); ok {
		applyIfNotSet("circuit-breaker-enabled", func() { cfg.CircuitBreakerEnabled = v })
	}

	if v, ok := getEnvInt("CB_FAILURE_THRESHOLD"); ok {
		applyIfNotSet("cb-failure-threshold", func() { cfg.CBFailureThreshold = v })
	}

	if v, ok := getEnvInt("CB_SUCCESS_THRESHOLD"); ok {
		applyIfNotSet("cb-success-threshold", func() { cfg.CBSuccessThreshold = v })
	}

	if v, ok := getEnvDuration("CB_TIMEOUT"); ok {
		applyIfNotSet("cb-timeout", func() { cfg.CBTimeout = v })
	}

	// Health checks
	if v, ok := getEnvBool("HEALTH_CHECK_ENABLED"); ok {
		applyIfNotSet("health-check-enabled", func() { cfg.HealthCheckEnabled = v })
	}

	if v, ok := getEnvString("HEALTH_CHECK_TYPE"); ok {
		applyIfNotSet("health-check-type", func() { cfg.HealthCheckType = v })
	}

	if v, ok := getEnvDuration("HEALTH_CHECK_INTERVAL"); ok {
		applyIfNotSet("health-check-interval", func() { cfg.HealthCheckInterval = v })
	}

	if v, ok := getEnvDuration("HEALTH_CHECK_TIMEOUT"); ok {
		applyIfNotSet("health-check-timeout", func() { cfg.HealthCheckTimeout = v })
	}

	if v, ok := getEnvString("HEALTH_CHECK_TARGET"); ok {
		applyIfNotSet("health-check-target", func() { cfg.HealthCheckTarget = v })
	}

	if v, ok := getEnvInt("HEALTH_CHECK_FAILURE_THRESHOLD"); ok {
		applyIfNotSet("health-check-failure-threshold", func() { cfg.HealthCheckFailureThreshold = v })
	}

	if v, ok := getEnvInt("HEALTH_CHECK_SUCCESS_THRESHOLD"); ok {
		applyIfNotSet("health-check-success-threshold", func() { cfg.HealthCheckSuccessThreshold = v })
	}
}
