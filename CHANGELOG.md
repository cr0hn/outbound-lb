# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.0] - 2026-01-31

### Fixed
- **CRITICAL**: Fixed TOCTOU race condition in limiter using CAS loops
- **CRITICAL**: Fixed race condition in CONNECT tunnel byte counting using atomic operations
- **CRITICAL**: Fixed io.Copy error handling in HTTP handler - errors now logged properly
- **HIGH**: Fixed getClientIP() logic inversion - now correctly extracts IP from RemoteAddr
- **HIGH**: Fixed timing attack vulnerability in authentication using constant-time comparison
- Added proper IPv6 address parsing support in getClientIP()

### Added
- Comprehensive stress tests for limiter with 10K+ goroutines
- Race condition detection tests for tunnel operations
- IPv6 address tests for client IP extraction
- CONTRIBUTING.md with development guidelines
- Improved README with table of contents, architecture diagram, and extensive documentation

### Changed
- Improved test coverage to ~80%
- Enhanced error logging in tunnel operations

## [1.0.0] - 2024-01-31

### Added
- Initial release
- HTTP/HTTPS forward proxy support
- CONNECT tunnel support for HTTPS
- LRU per-host load balancing algorithm
- Per-IP and total connection limiting
- Basic proxy authentication
- Prometheus metrics
- Health, readiness, and stats endpoints
- Structured logging (JSON/text)
- Graceful shutdown
- Docker support
- systemd service file
- GitHub Actions CI/CD
