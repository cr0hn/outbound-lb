# Java Demo

Demonstrates how to use Outbound LB proxy with Java.

## Requirements

- Java 11+

## Compiling and Running the Demo

```bash
# Compile
javac BasicProxy.java

# Run (make sure the proxy is running first)
java BasicProxy
```

Or using a single command (Java 11+):

```bash
java BasicProxy.java
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
5. **Concurrent Requests** - CompletableFuture for async parallel requests

## Quick Code Snippet

```java
import java.net.InetSocketAddress;
import java.net.ProxySelector;
import java.net.URI;
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
System.out.println(response.body());
```

## With Authentication

```java
import java.net.Authenticator;
import java.net.PasswordAuthentication;

Authenticator authenticator = new Authenticator() {
    @Override
    protected PasswordAuthentication getPasswordAuthentication() {
        if (getRequestorType() == RequestorType.PROXY) {
            return new PasswordAuthentication("user", "password".toCharArray());
        }
        return null;
    }
};

HttpClient client = HttpClient.newBuilder()
    .proxy(ProxySelector.of(new InetSocketAddress("localhost", 3128)))
    .authenticator(authenticator)
    .build();
```

## Using System Properties

```bash
java -Dhttp.proxyHost=localhost -Dhttp.proxyPort=3128 \
     -Dhttps.proxyHost=localhost -Dhttps.proxyPort=3128 \
     YourProgram
```
