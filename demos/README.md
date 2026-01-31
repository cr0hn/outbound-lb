# Outbound LB - Demos

This directory contains example code demonstrating how to use Outbound LB proxy in various programming languages.

## Prerequisites

Before running any demo, make sure Outbound LB is running:

```bash
# Start without authentication
outbound-lb --ips "YOUR_OUTBOUND_IP" --port 3128

# Start with authentication
outbound-lb --ips "YOUR_OUTBOUND_IP" --port 3128 --auth "user:password"
```

## Available Demos

| Language | Directory | Description |
|----------|-----------|-------------|
| [Python](python/) | `demos/python/` | Using `requests` and `aiohttp` |
| [Node.js](nodejs/) | `demos/nodejs/` | Using `axios` and `https-proxy-agent` |
| [Go](go/) | `demos/go/` | Using `net/http` with proxy transport |
| [Java](java/) | `demos/java/` | Using `HttpClient` with proxy selector |
| [Ruby](ruby/) | `demos/ruby/` | Using `net/http` with proxy |
| [PHP](php/) | `demos/php/` | Using cURL extension |
| [Rust](rust/) | `demos/rust/` | Using `reqwest` with proxy |
| [curl/bash](curl/) | `demos/curl/` | Shell scripts using curl |

## What Each Demo Covers

Each demo includes examples for:

1. **Basic HTTP request** through the proxy
2. **HTTPS request** (CONNECT tunnel)
3. **Authenticated proxy** connection
4. **Error handling** for common scenarios
5. **Concurrent requests** to demonstrate load balancing

## Quick Test

The fastest way to test your proxy setup:

```bash
# Test HTTP
curl -x http://localhost:3128 http://httpbin.org/ip

# Test HTTPS
curl -x http://localhost:3128 https://httpbin.org/ip

# Test with authentication
curl -x http://user:password@localhost:3128 http://httpbin.org/ip
```

## Configuration

All demos use these default values (configurable via environment variables):

| Variable | Default | Description |
|----------|---------|-------------|
| `PROXY_HOST` | `localhost` | Proxy hostname |
| `PROXY_PORT` | `3128` | Proxy port |
| `PROXY_USER` | `user` | Proxy username (for auth demos) |
| `PROXY_PASS` | `password` | Proxy password (for auth demos) |

## Verifying Load Balancing

To verify that requests are being distributed across multiple IPs:

1. Start the proxy with multiple outbound IPs
2. Run multiple requests to `http://httpbin.org/ip`
3. Observe different source IPs in the responses

Example with curl:
```bash
for i in {1..10}; do
  curl -s -x http://localhost:3128 http://httpbin.org/ip
  echo
done
```
