// Package config handles configuration parsing and hot reloading.
package config

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/cr0hn/outbound-lb/internal/logger"
)

// ConfigWatcher watches a configuration file for changes and notifies callbacks.
type ConfigWatcher struct {
	path      string
	current   atomic.Value // *Config
	watcher   *fsnotify.Watcher
	callbacks []func(*Config)
	stopCh    chan struct{}
	mu        sync.RWMutex
}

// NewConfigWatcher creates a new ConfigWatcher for the given config file path.
func NewConfigWatcher(path string, initial *Config) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	cw := &ConfigWatcher{
		path:    path,
		watcher: watcher,
		stopCh:  make(chan struct{}),
	}
	cw.current.Store(initial)

	return cw, nil
}

// Start begins watching the configuration file for changes.
func (w *ConfigWatcher) Start() error {
	if err := w.watcher.Add(w.path); err != nil {
		return err
	}

	go w.watchLoop()
	logger.Info("config_watcher_started", "path", w.path)
	return nil
}

// Stop stops the configuration watcher.
func (w *ConfigWatcher) Stop() {
	close(w.stopCh)
	w.watcher.Close()
	logger.Info("config_watcher_stopped")
}

// Current returns the current configuration.
func (w *ConfigWatcher) Current() *Config {
	return w.current.Load().(*Config)
}

// RegisterCallback adds a callback to be called when configuration changes.
func (w *ConfigWatcher) RegisterCallback(fn func(*Config)) {
	w.mu.Lock()
	w.callbacks = append(w.callbacks, fn)
	w.mu.Unlock()
}

// Reload manually reloads the configuration file.
func (w *ConfigWatcher) Reload() error {
	return w.reload()
}

// watchLoop watches for file changes with debouncing.
func (w *ConfigWatcher) watchLoop() {
	var debounceTimer *time.Timer
	debounceDuration := 100 * time.Millisecond

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Only react to write and create events
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				// Debounce: reset timer on each event
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(debounceDuration, func() {
					if err := w.reload(); err != nil {
						logger.Error("config_reload_failed", "error", err)
					}
				})
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			logger.Error("config_watcher_error", "error", err)

		case <-w.stopCh:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return
		}
	}
}

// reload loads the configuration from file and notifies callbacks.
func (w *ConfigWatcher) reload() error {
	newCfg, err := LoadFromFile(w.path)
	if err != nil {
		return err
	}

	// Validate the new configuration (only reloadable fields matter)
	if err := w.validateReloadable(newCfg); err != nil {
		return err
	}

	oldCfg := w.Current()
	w.current.Store(newCfg)

	// Log what changed
	w.logChanges(oldCfg, newCfg)

	// Notify callbacks
	w.mu.RLock()
	callbacks := make([]func(*Config), len(w.callbacks))
	copy(callbacks, w.callbacks)
	w.mu.RUnlock()

	for _, cb := range callbacks {
		cb(newCfg)
	}

	logger.Info("config_reloaded", "path", w.path)
	return nil
}

// validateReloadable validates only the hot-reloadable configuration fields.
func (w *ConfigWatcher) validateReloadable(cfg *Config) error {
	// Validate log level
	validLevels := map[string]bool{"trace": true, "debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[cfg.LogLevel] {
		return &ValidationError{Field: "log_level", Message: "must be trace, debug, info, warn, or error"}
	}

	// Validate log format
	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[cfg.LogFormat] {
		return &ValidationError{Field: "log_format", Message: "must be json or text"}
	}

	// Validate limits
	if cfg.MaxConnsPerIP < 1 {
		return &ValidationError{Field: "max_conns_per_ip", Message: "must be at least 1"}
	}
	if cfg.MaxConnsTotal < 1 {
		return &ValidationError{Field: "max_conns_total", Message: "must be at least 1"}
	}

	// Validate history settings
	if cfg.HistoryWindow <= 0 {
		return &ValidationError{Field: "history_window", Message: "must be positive"}
	}
	if cfg.HistorySize < 1 {
		return &ValidationError{Field: "history_size", Message: "must be at least 1"}
	}

	return nil
}

// logChanges logs which configuration values changed.
func (w *ConfigWatcher) logChanges(old, new *Config) {
	if old.LogLevel != new.LogLevel {
		logger.Info("config_changed", "field", "log_level", "old", old.LogLevel, "new", new.LogLevel)
	}
	if old.LogFormat != new.LogFormat {
		logger.Info("config_changed", "field", "log_format", "old", old.LogFormat, "new", new.LogFormat)
	}
	if old.MaxConnsPerIP != new.MaxConnsPerIP {
		logger.Info("config_changed", "field", "max_conns_per_ip", "old", old.MaxConnsPerIP, "new", new.MaxConnsPerIP)
	}
	if old.MaxConnsTotal != new.MaxConnsTotal {
		logger.Info("config_changed", "field", "max_conns_total", "old", old.MaxConnsTotal, "new", new.MaxConnsTotal)
	}
	if old.HistoryWindow != new.HistoryWindow {
		logger.Info("config_changed", "field", "history_window", "old", old.HistoryWindow, "new", new.HistoryWindow)
	}
	if old.HistorySize != new.HistorySize {
		logger.Info("config_changed", "field", "history_size", "old", old.HistorySize, "new", new.HistorySize)
	}

	// Warn about non-reloadable fields that changed
	if len(old.IPs) != len(new.IPs) || !slicesEqual(old.IPs, new.IPs) {
		logger.Warn("config_change_ignored", "field", "ips", "reason", "requires restart")
	}
	if old.Port != new.Port {
		logger.Warn("config_change_ignored", "field", "port", "reason", "requires restart")
	}
	if old.MetricsPort != new.MetricsPort {
		logger.Warn("config_change_ignored", "field", "metrics_port", "reason", "requires restart")
	}
	if old.Auth != new.Auth {
		logger.Warn("config_change_ignored", "field", "auth", "reason", "requires restart for security")
	}
	if old.Timeout != new.Timeout {
		logger.Warn("config_change_ignored", "field", "timeout", "reason", "requires restart")
	}
}

// slicesEqual compares two string slices for equality.
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
