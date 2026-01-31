# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2025-02-01

### Added
- HTTP/HTTPS forward proxy support
- CONNECT tunnel support for HTTPS
- LRU per-host load balancing algorithm
- Per-IP and total connection limiting
- Basic proxy authentication
- Prometheus metrics with 17 metric types:
  - `outbound_lb_requests_total`
  - `outbound_lb_request_duration_seconds`
  - `outbound_lb_bytes_sent_total`
  - `outbound_lb_bytes_received_total`
  - `outbound_lb_active_connections`
  - `outbound_lb_connections_per_ip`
  - `outbound_lb_balancer_selections_total`
  - `outbound_lb_limit_rejections_total`
  - `outbound_lb_auth_failures_total`
  - `outbound_lb_tunnel_connections_total`
  - `outbound_lb_history_entries`
  - `outbound_lb_history_hosts`
  - `outbound_lb_health_check_total`
  - `outbound_lb_ip_health_status`
  - `outbound_lb_health_check_duration_seconds`
  - `outbound_lb_healthy_ips`
  - `outbound_lb_unhealthy_ips`
- Health check system (TCP/HTTP) with configurable thresholds
- Health (`/health`), readiness (`/ready`), and stats (`/stats`) endpoints
- Structured logging (JSON/text formats)
- Hot configuration reload via SIGHUP
- Graceful shutdown with connection draining
- Docker support with multi-arch images (amd64, arm64, armv7)
- systemd service file
- GitHub Actions CI/CD with:
  - Automated testing with race detection
  - Code coverage verification (≥70%)
  - Multi-platform releases (35+ binaries)
  - Docker image publishing

### Security
- Fixed TOCTOU race condition in limiter using CAS loops
- Fixed race condition in CONNECT tunnel byte counting using atomic operations
- Fixed timing attack vulnerability in authentication using constant-time comparison
- Proper IPv6 address parsing support

### Platforms
Binary releases available for:
- **Linux**: amd64, arm64, arm (v5/v6/v7), 386, mips, mipsle, mips64, mips64le, ppc64, ppc64le, riscv64, s390x
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)
- **Windows**: amd64, arm64, 386
- **FreeBSD**: amd64, arm64, arm, 386, riscv64
- **OpenBSD**: amd64, arm64, 386
- **NetBSD**: amd64, arm64, arm, 386
