// Package main is the entry point for outbound-lb.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cr0hn/outbound-lb/internal/balancer"
	"github.com/cr0hn/outbound-lb/internal/config"
	"github.com/cr0hn/outbound-lb/internal/health"
	"github.com/cr0hn/outbound-lb/internal/limiter"
	"github.com/cr0hn/outbound-lb/internal/logger"
	"github.com/cr0hn/outbound-lb/internal/metrics"
	"github.com/cr0hn/outbound-lb/internal/proxy"
)

// Version information set via ldflags at build time.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	// Parse configuration
	cfg, err := config.ParseFlags()
	if err != nil {
		logger.Error("failed to parse configuration", "error", err)
		os.Exit(1)
	}

	// Initialize logger
	logger.Init(cfg.LogLevel, cfg.LogFormat)
	logger.Info("outbound-lb starting",
		"version", version,
		"commit", commit,
		"date", date,
		"ips", cfg.IPs,
		"port", cfg.Port,
		"metrics_port", cfg.MetricsPort,
	)

	// Create components
	stats := metrics.NewStatsCollector(cfg.IPs)
	lim := limiter.New(cfg.MaxConnsPerIP, cfg.MaxConnsTotal, cfg.IPs)

	// Create health checker if enabled
	var healthChecker *health.HealthChecker
	if cfg.HealthCheckEnabled {
		var checker health.Checker
		switch cfg.HealthCheckType {
		case "http":
			checker = health.NewHTTPChecker(cfg.HealthCheckTarget, cfg.HealthCheckTimeout)
			logger.Info("health_check_configured", "type", "http", "target", cfg.HealthCheckTarget)
		default:
			checker = health.NewTCPChecker(cfg.HealthCheckTarget, cfg.HealthCheckTimeout)
			logger.Info("health_check_configured", "type", "tcp", "target", cfg.HealthCheckTarget)
		}

		healthChecker = health.NewHealthChecker(health.HealthCheckerConfig{
			IPs:              cfg.IPs,
			Checker:          checker,
			Interval:         cfg.HealthCheckInterval,
			Timeout:          cfg.HealthCheckTimeout,
			FailureThreshold: cfg.HealthCheckFailureThreshold,
			SuccessThreshold: cfg.HealthCheckSuccessThreshold,
		})
		healthChecker.Start()
	}

	balCfg := balancer.Config{
		IPs:           cfg.IPs,
		HistoryWindow: int64(cfg.HistoryWindow.Seconds()),
		HistorySize:   cfg.HistorySize,
		Limiter:       lim,
		HealthChecker: healthChecker,
	}
	bal := balancer.New(balCfg)
	bal.Start()

	// Create servers
	proxyServer := proxy.NewServer(cfg, bal, lim, stats)
	metricsServer := metrics.NewServer(cfg.MetricsPort, stats)

	// Set up config watcher if config file is specified
	var cfgWatcher *config.ConfigWatcher
	if cfg.ConfigFile != "" {
		var watcherErr error
		cfgWatcher, watcherErr = config.NewConfigWatcher(cfg.ConfigFile, cfg)
		if watcherErr != nil {
			logger.Error("failed to create config watcher", "error", watcherErr)
		} else {
			// Register callback for configuration changes
			cfgWatcher.RegisterCallback(func(newCfg *config.Config) {
				// Reconfigure logger
				logger.Reconfigure(newCfg.LogLevel, newCfg.LogFormat)

				// Update limiter
				lim.UpdateLimits(newCfg.MaxConnsPerIP, newCfg.MaxConnsTotal)

				// Update balancer history config
				bal.UpdateHistoryConfig(newCfg.HistoryWindow, newCfg.HistorySize)
			})

			if startErr := cfgWatcher.Start(); startErr != nil {
				logger.Error("failed to start config watcher", "error", startErr)
			}
		}
	}

	// Start metrics server
	go func() {
		logger.Info("starting metrics server", "port", cfg.MetricsPort)
		if err := metricsServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("metrics server error", "error", err)
		}
	}()

	// Start proxy server
	go func() {
		metricsServer.SetReady(true)
		if err := proxyServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("proxy server error", "error", err)
			os.Exit(1)
		}
	}()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Wait for signals
	for {
		sig := <-sigCh

		// Handle SIGHUP for manual config reload
		if sig == syscall.SIGHUP {
			logger.Info("received SIGHUP, reloading configuration")
			if cfgWatcher != nil {
				if reloadErr := cfgWatcher.Reload(); reloadErr != nil {
					logger.Error("config reload failed", "error", reloadErr)
				}
			} else {
				logger.Warn("config reload requested but no config file specified")
			}
			continue
		}

		// SIGINT or SIGTERM - shutdown
		logger.Info("received shutdown signal", "signal", sig)
		break
	}

	// Graceful shutdown
	if cfgWatcher != nil {
		cfgWatcher.Stop()
	}

	metricsServer.SetReady(false)

	// Stop accepting new connections
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Wait for active connections
	logger.Info("waiting for active connections to complete")
	proxyServer.WaitForConnections(30 * time.Second)

	// Shutdown servers
	if err := proxyServer.Shutdown(ctx); err != nil {
		logger.Error("proxy server shutdown error", "error", err)
	}

	bal.Stop()

	// Stop health checker
	if healthChecker != nil {
		healthChecker.Stop()
	}

	if err := metricsServer.Shutdown(ctx); err != nil {
		logger.Error("metrics server shutdown error", "error", err)
	}

	logger.Info("outbound-lb stopped")
}
