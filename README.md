<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version">
  <img src="https://img.shields.io/github/license/cr0hn/outbound-lb?style=for-the-badge" alt="License">
  <img src="https://img.shields.io/github/v/release/cr0hn/outbound-lb?style=for-the-badge" alt="Release">
  <img src="https://img.shields.io/github/actions/workflow/status/cr0hn/outbound-lb/ci.yml?branch=main&style=for-the-badge&label=CI" alt="CI Status">
  <img src="https://img.shields.io/codecov/c/github/cr0hn/outbound-lb?style=for-the-badge" alt="Coverage">
</p>

<h1 align="center">Outbound LB</h1>

<p align="center">
  <strong>High-Performance Forward Proxy with Intelligent Outbound IP Load Balancing</strong>
</p>

<p align="center">
  A production-ready HTTP/HTTPS forward proxy that intelligently distributes client connections across multiple outbound IP addresses using an LRU per-host algorithm. Perfect for avoiding rate limits, IP-based restrictions, and maximizing throughput when interacting with external APIs.
</p>

---

## Table of Contents

- [Features](#features)
- [Use Cases](#use-cases)
- [Architecture](#architecture)
- [Installation](#installation)
  - [From Source](#from-source)
  - [Pre-built Binaries](#pre-built-binaries)
  - [Docker](#docker)
  - [Homebrew](#homebrew)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
  - [CLI Flags](#cli-flags)
  - [Configuration File (YAML)](#configuration-file-yaml)
  - [Environment Variables](#environment-variables)
- [Usage Examples](#usage-examples)
  - [Basic HTTP Proxy](#basic-http-proxy)
  - [HTTPS Tunneling (CONNECT)](#https-tunneling-connect)
  - [With Authentication](#with-authentication)
  - [Programming Languages](#programming-languages)
- [Load Balancing Algorithm](#load-balancing-algorithm)
- [IP Health Checks](#ip-health-checks)
- [Monitoring & Observability](#monitoring--observability)
  - [Health Endpoints](#health-endpoints)
  - [Prometheus Metrics](#prometheus-metrics)
  - [Grafana Dashboard](#grafana-dashboard)
- [Deployment](#deployment)
  - [Docker Compose](#docker-compose)
  - [Kubernetes](#kubernetes)
  - [Systemd](#systemd)
- [Security](#security)
- [Performance](#performance)
- [Development](#development)
  - [Prerequisites](#prerequisites)
  - [Building](#building)
  - [Testing](#testing)
  - [Contributing](#contributing)
- [Roadmap](#roadmap)
- [License](#license)

---

## Features

| Feature | Description |
|---------|-------------|
| **HTTP/HTTPS Proxy** | Full support for HTTP requests and CONNECT tunnels for HTTPS |
| **Intelligent Load Balancing** | LRU per-host algorithm for optimal IP distribution |
| **IP Health Checks** | Active TCP/HTTP probing with automatic failover |
| **Connection Limiting** | Per-IP and total connection limits to prevent overload |
| **Basic Authentication** | Optional proxy authentication for security |
| **Prometheus Metrics** | Full observability with detailed metrics |
| **Health Checks** | Liveness, readiness, and detailed stats endpoints |
| **Graceful Shutdown** | Clean connection draining on SIGTERM/SIGINT |
| **Structured Logging** | JSON or text format with configurable log levels |
| **Zero Dependencies** | Single binary with no external runtime dependencies |
| **Cross-Platform** | Builds for Linux, macOS, and Windows (amd64/arm64) |

---

## Use Cases

- **API Rate Limit Avoidance**: Distribute requests across multiple IPs to avoid per-IP rate limits
- **Web Scraping**: Rotate outbound IPs to prevent blocking
- **Load Distribution**: Balance outbound traffic across multiple network interfaces
- **High-Availability**: Use multiple egress points for redundancy
- **Testing**: Simulate requests from different source IPs

---

## Architecture

```
                                      ┌─────────────────┐
                                      │   Target API    │
                                      │  (example.com)  │
                                      └────────▲────────┘
                                               │
                    ┌──────────────────────────┼──────────────────────────┐
                    │                          │                          │
              ┌─────┴─────┐            ┌───────┴───────┐          ┌───────┴───────┐
              │  IP: .100 │            │   IP: .101    │          │   IP: .102    │
              └─────▲─────┘            └───────▲───────┘          └───────▲───────┘
                    │                          │                          │
                    └──────────────────────────┼──────────────────────────┘
                                               │
                                    ┌──────────┴──────────┐
                                    │                     │
                                    │    Outbound LB      │
                                    │                     │
                                    │  ┌───────────────┐  │
                                    │  │ LRU Balancer  │  │
                                    │  │  (per host)   │  │
                                    │  └───────────────┘  │
                                    │                     │
                                    │  ┌───────────────┐  │
                                    │  │   Limiter     │  │
                                    │  │ (per-IP/total)│  │
                                    │  └───────────────┘  │
                                    │                     │
                                    └──────────▲──────────┘
                                               │
                              ┌────────────────┼────────────────┐
                              │                │                │
                        ┌─────┴─────┐    ┌─────┴─────┐    ┌─────┴─────┐
                        │  Client 1 │    │  Client 2 │    │  Client N │
                        └───────────┘    └───────────┘    └───────────┘
```

---

## Installation

### From Source

```bash
go install github.com/cr0hn/outbound-lb/cmd/outbound-lb@latest
```

### Pre-built Binaries

Download the latest release for your platform from the [Releases page](https://github.com/cr0hn/outbound-lb/releases).

```bash
# Linux (amd64)
curl -LO https://github.com/cr0hn/outbound-lb/releases/latest/download/outbound-lb_linux_amd64.tar.gz
tar -xzf outbound-lb_linux_amd64.tar.gz
sudo mv outbound-lb /usr/local/bin/

# macOS (Apple Silicon)
curl -LO https://github.com/cr0hn/outbound-lb/releases/latest/download/outbound-lb_darwin_arm64.tar.gz
tar -xzf outbound-lb_darwin_arm64.tar.gz
sudo mv outbound-lb /usr/local/bin/
```

### Docker

```bash
# Pull from Docker Hub
docker pull cr0hn/outbound-lb:latest

# Run with Docker
docker run -d \
  --name outbound-lb \
  -p 3128:3128 \
  -p 9090:9090 \
  cr0hn/outbound-lb:latest \
  --ips "192.168.1.100,192.168.1.101"
```

### Homebrew

```bash
# Coming soon
brew install cr0hn/tap/outbound-lb
```

---

## Quick Start

```bash
# Basic usage with single outbound IP
outbound-lb --ips "192.168.1.100" --port 3128

# Multiple outbound IPs for load balancing
outbound-lb --ips "192.168.1.100,192.168.1.101,192.168.1.102" --port 3128

# With authentication enabled
outbound-lb --ips "192.168.1.100" --auth "user:password"

# Debug mode with human-readable logs
outbound-lb --ips "192.168.1.100" --log-level debug --log-format text
```

Test it:

```bash
# HTTP request through proxy
curl -x http://localhost:3128 http://ifconfig.me

# HTTPS request through proxy
curl -x http://localhost:3128 https://api.github.com
```

---

## Configuration

Configuration can be provided via CLI flags, environment variables, or YAML file. The order of precedence is:

1. **CLI flags** (highest priority)
2. **Environment variables**
3. **YAML config file**
4. **Defaults** (lowest priority)

### CLI Flags

#### Server Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `--ips` | *required* | Comma-separated list of outbound IPs |
| `--port` | `3128` | Proxy listening port |
| `--metrics-port` | `9090` | Metrics/health server port |
| `--auth` | - | Basic auth credentials (`user:pass`) |
| `--config` | - | Path to YAML config file |

#### Timeouts

| Flag | Default | Description |
|------|---------|-------------|
| `--timeout` | `30s` | Connection timeout |
| `--idle-timeout` | `60s` | Idle connection timeout |

#### Connection Limits

| Flag | Default | Description |
|------|---------|-------------|
| `--max-conns-per-ip` | `100` | Max concurrent connections per outbound IP |
| `--max-conns-total` | `1000` | Max total concurrent connections |

#### Load Balancer Settings

| Flag | Default | Description |
|------|---------|-------------|
| `--history-window` | `5m` | LRU history time window |
| `--history-size` | `100` | Max history entries per host |
| `--history-max-total-entries` | `100000` | Max total history entries across all hosts |

#### Transport Tuning

| Flag | Default | Description |
|------|---------|-------------|
| `--tcp-keepalive` | `30s` | TCP keep-alive interval |
| `--idle-conn-timeout` | `90s` | Idle HTTP connection timeout |
| `--tls-handshake-timeout` | `10s` | TLS handshake timeout |
| `--expect-continue-timeout` | `1s` | Expect-continue timeout |

#### Circuit Breaker

| Flag | Default | Description |
|------|---------|-------------|
| `--circuit-breaker-enabled` | `false` | Enable circuit breaker per IP |
| `--cb-failure-threshold` | `5` | Number of failures before opening the circuit |
| `--cb-success-threshold` | `2` | Number of successes in half-open to close the circuit |
| `--cb-timeout` | `30s` | How long the circuit stays open before half-open |

#### Health Checks

| Flag | Default | Description |
|------|---------|-------------|
| `--health-check-enabled` | `false` | Enable active health checks |
| `--health-check-type` | `tcp` | Health check type: `tcp` or `http` |
| `--health-check-interval` | `10s` | Interval between health checks |
| `--health-check-timeout` | `5s` | Timeout per health check |
| `--health-check-target` | `1.1.1.1:443` | Target for checks (host:port for TCP, URL for HTTP) |
| `--health-check-failure-threshold` | `3` | Consecutive failures before marking IP unhealthy |
| `--health-check-success-threshold` | `2` | Consecutive successes before marking IP healthy |

#### Logging

| Flag | Default | Description |
|------|---------|-------------|
| `--log-level` | `info` | Log level (`trace`, `debug`, `info`, `warn`, `error`) |
| `--log-format` | `json` | Log format (`json`, `text`) |

### Configuration File (YAML)

```yaml
# /etc/outbound-lb/config.yaml

# Required: List of outbound IP addresses
ips:
  - 192.168.1.100
  - 192.168.1.101
  - 192.168.1.102

# Server configuration
port: 3128
metrics_port: 9090

# Authentication (optional)
auth: "user:password"

# Timeouts
timeout: 30s
idle_timeout: 60s

# Connection limits
max_conns_per_ip: 100
max_conns_total: 1000

# Load balancer settings
history_window: 5m
history_size: 100
history_max_total_entries: 100000

# Transport tuning
tcp_keepalive: 30s
idle_conn_timeout: 90s
tls_handshake_timeout: 10s
expect_continue_timeout: 1s

# Circuit breaker
circuit_breaker_enabled: false
cb_failure_threshold: 5
cb_success_threshold: 2
cb_timeout: 30s

# Health checks
health_check_enabled: false
health_check_type: tcp
health_check_interval: 10s
health_check_timeout: 5s
health_check_target: "1.1.1.1:443"
health_check_failure_threshold: 3
health_check_success_threshold: 2

# Logging
log_level: info
log_format: json
```

Run with config file:

```bash
outbound-lb --config /etc/outbound-lb/config.yaml
```

> **Note**: CLI flags override config file and environment variable values.

### Environment Variables

All configuration options can be set via environment variables with the `OUTBOUND_LB_` prefix:

| Environment Variable | CLI Flag | Default |
|---------------------|----------|---------|
| `OUTBOUND_LB_IPS` | `--ips` | *required* |
| `OUTBOUND_LB_PORT` | `--port` | `3128` |
| `OUTBOUND_LB_METRICS_PORT` | `--metrics-port` | `9090` |
| `OUTBOUND_LB_AUTH` | `--auth` | - |
| `OUTBOUND_LB_TIMEOUT` | `--timeout` | `30s` |
| `OUTBOUND_LB_IDLE_TIMEOUT` | `--idle-timeout` | `60s` |
| `OUTBOUND_LB_MAX_CONNS_PER_IP` | `--max-conns-per-ip` | `100` |
| `OUTBOUND_LB_MAX_CONNS_TOTAL` | `--max-conns-total` | `1000` |
| `OUTBOUND_LB_HISTORY_WINDOW` | `--history-window` | `5m` |
| `OUTBOUND_LB_HISTORY_SIZE` | `--history-size` | `100` |
| `OUTBOUND_LB_HISTORY_MAX_TOTAL_ENTRIES` | `--history-max-total-entries` | `100000` |
| `OUTBOUND_LB_TCP_KEEPALIVE` | `--tcp-keepalive` | `30s` |
| `OUTBOUND_LB_IDLE_CONN_TIMEOUT` | `--idle-conn-timeout` | `90s` |
| `OUTBOUND_LB_TLS_HANDSHAKE_TIMEOUT` | `--tls-handshake-timeout` | `10s` |
| `OUTBOUND_LB_EXPECT_CONTINUE_TIMEOUT` | `--expect-continue-timeout` | `1s` |
| `OUTBOUND_LB_CIRCUIT_BREAKER_ENABLED` | `--circuit-breaker-enabled` | `false` |
| `OUTBOUND_LB_CB_FAILURE_THRESHOLD` | `--cb-failure-threshold` | `5` |
| `OUTBOUND_LB_CB_SUCCESS_THRESHOLD` | `--cb-success-threshold` | `2` |
| `OUTBOUND_LB_CB_TIMEOUT` | `--cb-timeout` | `30s` |
| `OUTBOUND_LB_HEALTH_CHECK_ENABLED` | `--health-check-enabled` | `false` |
| `OUTBOUND_LB_HEALTH_CHECK_TYPE` | `--health-check-type` | `tcp` |
| `OUTBOUND_LB_HEALTH_CHECK_INTERVAL` | `--health-check-interval` | `10s` |
| `OUTBOUND_LB_HEALTH_CHECK_TIMEOUT` | `--health-check-timeout` | `5s` |
| `OUTBOUND_LB_HEALTH_CHECK_TARGET` | `--health-check-target` | `1.1.1.1:443` |
| `OUTBOUND_LB_HEALTH_CHECK_FAILURE_THRESHOLD` | `--health-check-failure-threshold` | `3` |
| `OUTBOUND_LB_HEALTH_CHECK_SUCCESS_THRESHOLD` | `--health-check-success-threshold` | `2` |
| `OUTBOUND_LB_LOG_LEVEL` | `--log-level` | `info` |
| `OUTBOUND_LB_LOG_FORMAT` | `--log-format` | `json` |

Example:

```bash
export OUTBOUND_LB_IPS="192.168.1.100,192.168.1.101"
export OUTBOUND_LB_PORT=3128
export OUTBOUND_LB_AUTH="user:password"
export OUTBOUND_LB_LOG_LEVEL=debug
export OUTBOUND_LB_CIRCUIT_BREAKER_ENABLED=true

outbound-lb
```

---

## Usage Examples

### Basic HTTP Proxy

```bash
# Using curl
curl -x http://localhost:3128 http://httpbin.org/ip

# Using wget
wget -e http_proxy=http://localhost:3128 http://httpbin.org/ip

# Using HTTPie
http --proxy http:http://localhost:3128 httpbin.org/ip
```

### HTTPS Tunneling (CONNECT)

```bash
# HTTPS through CONNECT tunnel
curl -x http://localhost:3128 https://api.github.com/zen

# Verbose mode to see the CONNECT handshake
curl -v -x http://localhost:3128 https://httpbin.org/get
```

### With Authentication

```bash
# Credentials in URL
curl -x http://user:password@localhost:3128 http://httpbin.org/ip

# Using Proxy-Authorization header
curl -x http://localhost:3128 \
  -H "Proxy-Authorization: Basic $(echo -n 'user:password' | base64)" \
  http://httpbin.org/ip
```

### Programming Languages

<details>
<summary><b>Python</b></summary>

```python
import requests

proxies = {
    'http': 'http://localhost:3128',
    'https': 'http://localhost:3128',
}

# Without authentication
response = requests.get('https://api.example.com', proxies=proxies)

# With authentication
proxies_auth = {
    'http': 'http://user:password@localhost:3128',
    'https': 'http://user:password@localhost:3128',
}
response = requests.get('https://api.example.com', proxies=proxies_auth)
```

</details>

<details>
<summary><b>Node.js</b></summary>

```javascript
const axios = require('axios');
const { HttpsProxyAgent } = require('https-proxy-agent');

const agent = new HttpsProxyAgent('http://localhost:3128');

axios.get('https://api.example.com', {
  httpsAgent: agent,
  proxy: false  // Important: disable axios's default proxy handling
});
```

</details>

<details>
<summary><b>Go</b></summary>

```go
package main

import (
    "net/http"
    "net/url"
)

func main() {
    proxyURL, _ := url.Parse("http://localhost:3128")

    client := &http.Client{
        Transport: &http.Transport{
            Proxy: http.ProxyURL(proxyURL),
        },
    }

    resp, err := client.Get("https://api.example.com")
    // handle response
}
```

</details>

<details>
<summary><b>Java</b></summary>

```java
import java.net.InetSocketAddress;
import java.net.ProxySelector;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;

HttpClient client = HttpClient.newBuilder()
    .proxy(ProxySelector.of(new InetSocketAddress("localhost", 3128)))
    .build();

HttpRequest request = HttpRequest.newBuilder()
    .uri(URI.create("https://api.example.com"))
    .build();

HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());
```

</details>

---

## Demos

The `demos/` directory contains example code for using Outbound LB in various programming languages:

| Language | Directory | Description |
|----------|-----------|-------------|
| [Python](demos/python/) | `demos/python/` | Using `requests` and `aiohttp` |
| [Node.js](demos/nodejs/) | `demos/nodejs/` | Using `axios` and `https-proxy-agent` |
| [Go](demos/go/) | `demos/go/` | Using `net/http` with proxy transport |
| [Java](demos/java/) | `demos/java/` | Using `HttpClient` with proxy selector |
| [Ruby](demos/ruby/) | `demos/ruby/` | Using `net/http` with proxy |
| [PHP](demos/php/) | `demos/php/` | Using cURL extension |
| [Rust](demos/rust/) | `demos/rust/` | Using `reqwest` with proxy |
| [curl/bash](demos/curl/) | `demos/curl/` | Shell scripts using curl |

Each demo includes examples for:
- Basic HTTP/HTTPS requests through the proxy
- Proxy authentication
- Error handling
- Concurrent requests to demonstrate load balancing

See the [demos/README.md](demos/README.md) for more details.

---

## Hot Reload

Outbound LB supports hot reloading of certain configuration values without restarting the process.

### Hot-Reloadable Settings

| Setting | Reloadable | Notes |
|---------|------------|-------|
| `log_level` | Yes | Changes take effect immediately |
| `log_format` | Yes | Handler is recreated |
| `max_conns_per_ip` | Yes | Uses atomic operations |
| `max_conns_total` | Yes | Uses atomic operations |
| `history_window` | Yes | Affects new selections |
| `history_size` | Yes | Affects new selections |
| `ips` | No | Requires restart |
| `port` | No | Requires socket rebind |
| `metrics_port` | No | Requires socket rebind |
| `auth` | No | Security: requires restart |
| `timeout` | No | Affects existing connections |

### How to Reload

**Automatic**: Edit the configuration file while the proxy is running. Changes are detected automatically via filesystem events (with 100ms debounce).

**Manual**: Send SIGHUP to the process:
```bash
kill -HUP $(pidof outbound-lb)
# or
kill -HUP $(pgrep outbound-lb)
```

### Behavior

- Invalid configurations are rejected; the previous configuration is kept
- A log message confirms successful reload: `config_reloaded`
- Changes to non-reloadable fields log a warning but are ignored
- Multiple rapid file changes are debounced (100ms)

---

## Logging Levels

| Level | Description |
|-------|-------------|
| `error` | Critical errors that affect operation |
| `warn` | Warnings (limits reached, auth failures) |
| `info` | Normal operation (startup, shutdown, requests) |
| `debug` | Debugging info (IP selection, balancer decisions) |
| `trace` | Very detailed tracing (acquire/release, tunnel bytes, every step) |

> **Note**: `trace` level generates high log volume. Use only for troubleshooting specific issues.

### Example: Enabling Trace Logging

```bash
# Via CLI
outbound-lb --ips "192.168.1.100" --log-level trace

# Via config file
log_level: trace

# At runtime (edit config.yaml while running)
sed -i 's/log_level: info/log_level: trace/' config.yaml

# Or via SIGHUP after editing
kill -HUP $(pidof outbound-lb)
```

---

## Load Balancing Algorithm

The **LRU per-host** algorithm ensures optimal distribution of requests:

```
┌─────────────────────────────────────────────────────────────┐
│                    IP Selection Process                      │
├─────────────────────────────────────────────────────────────┤
│  1. Get history for destination host within time window     │
│     └─> "api.example.com" → last 5 minutes of selections    │
│                                                             │
│  2. Filter by max history size (most recent entries)        │
│     └─> Keep last 100 entries                               │
│                                                             │
│  3. Count usage per IP in filtered history                  │
│     └─> IP .100: 15 uses, IP .101: 12 uses, IP .102: 18     │
│                                                             │
│  4. Exclude IPs at connection limit                         │
│     └─> If IP .100 has 100 active conns, skip it            │
│                                                             │
│  5. Select IP with lowest usage count                       │
│     └─> IP .101 (12 uses) selected                          │
│                                                             │
│  6. Tie-break by oldest last-use timestamp                  │
│     └─> If tie, prefer IP not used recently                 │
│                                                             │
│  7. Record selection in history                             │
│     └─> Add: {host: "api.example.com", ip: ".101", time: X} │
└─────────────────────────────────────────────────────────────┘
```

---

## IP Health Checks

Outbound LB supports active health checking of outbound IPs with automatic failover.

### Health Check Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `--health-check-enabled` | `false` | Enable active health checks |
| `--health-check-type` | `tcp` | Check type: `tcp` or `http` |
| `--health-check-interval` | `10s` | Interval between checks |
| `--health-check-timeout` | `5s` | Timeout per check |
| `--health-check-target` | `1.1.1.1:443` | Target for checks (host:port for TCP, URL for HTTP) |
| `--health-check-failure-threshold` | `3` | Consecutive failures before unhealthy |
| `--health-check-success-threshold` | `2` | Consecutive successes before healthy |

### YAML Configuration

```yaml
# Enable health checks
health_check_enabled: true
health_check_type: tcp
health_check_interval: 10s
health_check_timeout: 5s
health_check_target: "1.1.1.1:443"
health_check_failure_threshold: 3
health_check_success_threshold: 2
```

### How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                    Health Check Flow                         │
├─────────────────────────────────────────────────────────────┤
│  1. Health checker probes each outbound IP periodically     │
│     └─> TCP connect or HTTP GET to target                   │
│                                                             │
│  2. Track consecutive failures per IP                       │
│     └─> IP .100: 0 failures, IP .101: 2 failures            │
│                                                             │
│  3. After N consecutive failures, mark IP unhealthy         │
│     └─> IP .101 marked unhealthy after 3 failures           │
│                                                             │
│  4. Unhealthy IPs excluded from load balancing              │
│     └─> Balancer only considers healthy IPs                 │
│                                                             │
│  5. After M consecutive successes, IP recovers              │
│     └─> IP .101 returns to healthy after 2 successes        │
│                                                             │
│  6. Graceful degradation if all IPs unhealthy               │
│     └─> Falls back to using all IPs                         │
└─────────────────────────────────────────────────────────────┘
```

### Health Check Types

**TCP Check** (default):
- Attempts TCP connection to target
- Success: connection established
- Use for basic connectivity verification

```bash
--health-check-type tcp --health-check-target "1.1.1.1:443"
```

**HTTP Check**:
- Performs HTTP GET request
- Success: 2xx or 3xx response
- Use when HTTP-level verification is needed

```bash
--health-check-type http --health-check-target "http://httpbin.org/status/200"
```

### Health Check Metrics

```promql
# Health check results counter
outbound_lb_health_check_total{ip="192.168.1.100", result="success"}
outbound_lb_health_check_total{ip="192.168.1.100", result="failure"}

# Current health status (1=healthy, 0=unhealthy)
outbound_lb_ip_health_status{ip="192.168.1.100"}

# Health check duration
outbound_lb_health_check_duration_seconds{ip="192.168.1.100"}

# Aggregate counts
outbound_lb_healthy_ips
outbound_lb_unhealthy_ips
```

---

## Monitoring & Observability

### Health Endpoints

| Endpoint | Port | Description |
|----------|------|-------------|
| `/health` | 9090 | Liveness probe - always returns 200 if server is running |
| `/ready` | 9090 | Readiness probe - returns 200 when ready to accept traffic |
| `/stats` | 9090 | JSON statistics including connections, requests, bytes |
| `/metrics` | 9090 | Prometheus metrics endpoint |

### Prometheus Metrics

```promql
# Request metrics
outbound_lb_requests_total{method="GET", status="200"}
outbound_lb_request_duration_seconds_bucket{method="GET", le="0.5"}

# Connection metrics
outbound_lb_active_connections
outbound_lb_connections_per_ip{ip="192.168.1.100"}
outbound_lb_tunnel_connections_total

# Load balancer metrics
outbound_lb_balancer_selections_total{ip="192.168.1.100", host="api.example.com"}

# Error metrics
outbound_lb_limit_rejections_total{type="per_ip"}
outbound_lb_auth_failures_total
```

### Grafana Dashboard

Import our pre-built Grafana dashboard for comprehensive monitoring:

```bash
# Dashboard ID: coming-soon
# Or import from: deployments/grafana/dashboard.json
```

---

## Deployment

### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  outbound-lb:
    image: cr0hn/outbound-lb:latest
    ports:
      - "3128:3128"
      - "9090:9090"
    command:
      - --ips=192.168.1.100,192.168.1.101
      - --auth=user:password
      - --log-level=info
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:9090/health"]
      interval: 10s
      timeout: 5s
      retries: 3

  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9091:9090"
```

### Kubernetes

```yaml
# kubernetes/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: outbound-lb
spec:
  replicas: 2
  selector:
    matchLabels:
      app: outbound-lb
  template:
    metadata:
      labels:
        app: outbound-lb
    spec:
      containers:
      - name: outbound-lb
        image: cr0hn/outbound-lb:latest
        args:
          - --ips=$(OUTBOUND_IPS)
          - --auth=$(PROXY_AUTH)
        env:
          - name: OUTBOUND_IPS
            valueFrom:
              configMapKeyRef:
                name: outbound-lb-config
                key: ips
          - name: PROXY_AUTH
            valueFrom:
              secretKeyRef:
                name: outbound-lb-secret
                key: auth
        ports:
          - containerPort: 3128
            name: proxy
          - containerPort: 9090
            name: metrics
        livenessProbe:
          httpGet:
            path: /health
            port: metrics
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: metrics
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          limits:
            cpu: "1"
            memory: "256Mi"
          requests:
            cpu: "100m"
            memory: "64Mi"
```

### Systemd

```ini
# /etc/systemd/system/outbound-lb.service
[Unit]
Description=Outbound Load Balancer Proxy
After=network.target

[Service]
Type=simple
User=nobody
Group=nogroup
ExecStart=/usr/local/bin/outbound-lb --config /etc/outbound-lb/config.yaml
Restart=always
RestartSec=5

# Security hardening
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable outbound-lb
sudo systemctl start outbound-lb
```

---

## Security

### Best Practices

- **Always use authentication** in production environments
- **Use TLS termination** (nginx/haproxy) in front for encrypted proxy traffic
- **Restrict network access** to the proxy port using firewall rules
- **Rotate credentials** regularly
- **Monitor auth failures** via `outbound_lb_auth_failures_total` metric

### Security Features

- **Constant-time password comparison** to prevent timing attacks
- **Connection limits** to prevent resource exhaustion
- **No secrets in logs** - credentials are never logged
- **Minimal privileges** - runs as non-root user in Docker

---

## Performance

### Benchmarks

| Metric | Value |
|--------|-------|
| Requests/sec (HTTP) | ~50,000 |
| Requests/sec (HTTPS CONNECT) | ~30,000 |
| Memory usage (idle) | ~10 MB |
| Memory usage (10k connections) | ~100 MB |
| Startup time | < 100ms |

### Tuning Tips

```bash
# Increase file descriptor limits
ulimit -n 65535

# Optimize for high throughput
outbound-lb \
  --ips "..." \
  --max-conns-per-ip 500 \
  --max-conns-total 5000 \
  --timeout 60s \
  --idle-timeout 120s
```

---

## Development

### Prerequisites

- Go 1.21+
- Make
- Docker (optional)
- golangci-lint (for linting)

### Building

```bash
# Clone the repository
git clone https://github.com/cr0hn/outbound-lb.git
cd outbound-lb

# Build
make build

# Build for all platforms
make build-linux

# Run locally
make run
```

### Testing

```bash
# Run all tests
make test

# Run tests with race detector
go test -race ./...

# Run tests with coverage
make coverage

# Generate HTML coverage report
make coverage-html
```

### Contributing

Contributions are welcome! Please read our [Contributing Guidelines](CONTRIBUTING.md) before submitting a PR.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## Roadmap

- [ ] **SOCKS5 Support** - Add SOCKS5 proxy protocol support
- [ ] **Weighted Load Balancing** - Assign weights to outbound IPs
- [x] **IP Health Checks** - Automatic failover for unhealthy IPs
- [ ] **TLS Client Certificates** - Mutual TLS authentication
- [ ] **Request/Response Modification** - Header manipulation
- [ ] **Access Control Lists** - Allow/deny lists for destinations
- [ ] **Web UI** - Dashboard for monitoring and configuration

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<p align="center">
  <sub>Built with ❤️ by <a href="https://github.com/cr0hn">cr0hn</a></sub>
</p>
