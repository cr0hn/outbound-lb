# curl/bash Demo

Demonstrates how to use Outbound LB proxy with curl and other shell tools.

## Requirements

- curl
- bash 4+
- (optional) wget
- (optional) httpie

## Running the Demo

```bash
# Make sure the proxy is running first
chmod +x examples.sh
./examples.sh
```

## Configuration

Set these environment variables to customize the proxy connection:

```bash
export PROXY_HOST=localhost
export PROXY_PORT=3128
export PROXY_USER=user
export PROXY_PASS=password
```

## Examples Included

1. **Basic HTTP Request** - Simple GET request
2. **HTTPS Request** - CONNECT tunnel for secure connections
3. **Authenticated Proxy** - Using proxy credentials
4. **Verbose Output** - See CONNECT handshake
5. **Error Handling** - Timeouts, invalid URLs, auth failures
6. **Concurrent Requests** - Parallel requests in background
7. **Using wget** - Alternative HTTP client
8. **Using HTTPie** - Modern HTTP client
9. **POST Request** - Sending data through proxy
10. **Custom Headers** - Adding headers to requests

## Quick Reference

### Basic HTTP

```bash
curl -x http://localhost:3128 http://httpbin.org/ip
```

### HTTPS (CONNECT tunnel)

```bash
curl -x http://localhost:3128 https://httpbin.org/ip
```

### With Authentication

```bash
# Using -U flag
curl -x http://localhost:3128 -U user:password https://httpbin.org/ip

# Using URL syntax
curl -x http://user:password@localhost:3128 https://httpbin.org/ip
```

### Verbose Output

```bash
curl -v -x http://localhost:3128 https://httpbin.org/ip
```

### With Timeout

```bash
curl -x http://localhost:3128 --connect-timeout 5 -m 30 https://api.example.com
```

### POST Request

```bash
curl -x http://localhost:3128 \
  -X POST \
  -d '{"key":"value"}' \
  -H 'Content-Type: application/json' \
  https://api.example.com/endpoint
```

### Using Environment Variables

```bash
export http_proxy=http://localhost:3128
export https_proxy=http://localhost:3128
export HTTP_PROXY=http://localhost:3128
export HTTPS_PROXY=http://localhost:3128

# Now curl will use the proxy automatically
curl https://api.example.com
```

## Using wget

```bash
# Using environment variables
export http_proxy=http://localhost:3128
wget http://httpbin.org/ip

# Using -e option
wget -e http_proxy=http://localhost:3128 http://httpbin.org/ip
```

## Using HTTPie

```bash
# Install first: pip install httpie
http --proxy http:http://localhost:3128 httpbin.org/ip
```

## Load Balancing Test

```bash
# Make 10 requests and observe IP distribution
for i in {1..10}; do
  curl -s -x http://localhost:3128 http://httpbin.org/ip
done
```
