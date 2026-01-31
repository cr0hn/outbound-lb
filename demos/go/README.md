# Go Demo

Demonstrates how to use Outbound LB proxy with Go.

## Requirements

- Go 1.21+

## Running the Demo

```bash
# Make sure the proxy is running first
go run basic_proxy.go
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

1. **Basic HTTP Request** - Simple GET request through proxy
2. **HTTPS Request** - CONNECT tunnel for secure connections
3. **Authenticated Proxy** - Using proxy credentials
4. **Error Handling** - Timeouts, connection errors, auth failures
5. **Concurrent Requests** - Goroutines for parallel requests
6. **Context Cancellation** - Using context for request control

## Quick Code Snippet

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

## With Authentication

```go
proxyURL, _ := url.Parse("http://user:password@localhost:3128")

client := &http.Client{
    Transport: &http.Transport{
        Proxy: http.ProxyURL(proxyURL),
    },
}
```

## Using Environment Variable

Go's http package respects the `HTTP_PROXY` and `HTTPS_PROXY` environment variables:

```bash
export HTTP_PROXY=http://localhost:3128
export HTTPS_PROXY=http://localhost:3128
go run yourprogram.go
```

```go
// This will automatically use HTTP_PROXY/HTTPS_PROXY
client := &http.Client{
    Transport: &http.Transport{
        Proxy: http.ProxyFromEnvironment,
    },
}
```
