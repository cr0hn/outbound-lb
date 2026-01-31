/**
 * Outbound LB - Java Demo
 *
 * Demonstrates how to use Outbound LB proxy with Java using HttpClient.
 * Includes examples for HTTP, HTTPS, authentication, error handling, and concurrent requests.
 *
 * Requirements:
 *     Java 11+
 *
 * Usage:
 *     javac BasicProxy.java
 *     java BasicProxy
 */

import java.net.Authenticator;
import java.net.InetSocketAddress;
import java.net.PasswordAuthentication;
import java.net.ProxySelector;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.net.http.HttpTimeoutException;
import java.time.Duration;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

public class BasicProxy {

    // Configuration from environment variables
    private static final String PROXY_HOST = System.getenv().getOrDefault("PROXY_HOST", "localhost");
    private static final int PROXY_PORT = Integer.parseInt(System.getenv().getOrDefault("PROXY_PORT", "3128"));
    private static final String PROXY_USER = System.getenv().getOrDefault("PROXY_USER", "user");
    private static final String PROXY_PASS = System.getenv().getOrDefault("PROXY_PASS", "password");

    public static void main(String[] args) {
        System.out.println();
        System.out.println("Outbound LB - Java Demo");
        System.out.printf("Proxy: http://%s:%d%n", PROXY_HOST, PROXY_PORT);
        System.out.println();

        exampleHttpRequest();
        exampleHttpsRequest();
        exampleAuthenticatedProxy();
        exampleErrorHandling();
        exampleConcurrentRequests();

        System.out.println("All examples completed!");
    }

    private static void printSeparator(String title) {
        System.out.println("=".repeat(60));
        System.out.println(title);
        System.out.println("=".repeat(60));
    }

    /**
     * Example 1: Basic HTTP Request
     */
    private static void exampleHttpRequest() {
        printSeparator("Example 1: Basic HTTP Request");

        try {
            HttpClient client = HttpClient.newBuilder()
                    .proxy(ProxySelector.of(new InetSocketAddress(PROXY_HOST, PROXY_PORT)))
                    .connectTimeout(Duration.ofSeconds(10))
                    .build();

            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create("http://httpbin.org/ip"))
                    .timeout(Duration.ofSeconds(10))
                    .GET()
                    .build();

            HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());

            System.out.println("Status: " + response.statusCode());
            System.out.println("Response: " + response.body());
        } catch (Exception e) {
            System.out.println("Error: " + e.getMessage());
        }

        System.out.println();
    }

    /**
     * Example 2: HTTPS Request (CONNECT tunnel)
     */
    private static void exampleHttpsRequest() {
        printSeparator("Example 2: HTTPS Request (CONNECT tunnel)");

        try {
            HttpClient client = HttpClient.newBuilder()
                    .proxy(ProxySelector.of(new InetSocketAddress(PROXY_HOST, PROXY_PORT)))
                    .connectTimeout(Duration.ofSeconds(10))
                    .build();

            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create("https://httpbin.org/ip"))
                    .timeout(Duration.ofSeconds(10))
                    .GET()
                    .build();

            HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());

            System.out.println("Status: " + response.statusCode());
            System.out.println("Response: " + response.body());
        } catch (Exception e) {
            System.out.println("Error: " + e.getMessage());
        }

        System.out.println();
    }

    /**
     * Example 3: Authenticated Proxy
     */
    private static void exampleAuthenticatedProxy() {
        printSeparator("Example 3: Authenticated Proxy");

        try {
            // Create an authenticator for proxy auth
            Authenticator authenticator = new Authenticator() {
                @Override
                protected PasswordAuthentication getPasswordAuthentication() {
                    if (getRequestorType() == RequestorType.PROXY) {
                        return new PasswordAuthentication(PROXY_USER, PROXY_PASS.toCharArray());
                    }
                    return null;
                }
            };

            HttpClient client = HttpClient.newBuilder()
                    .proxy(ProxySelector.of(new InetSocketAddress(PROXY_HOST, PROXY_PORT)))
                    .authenticator(authenticator)
                    .connectTimeout(Duration.ofSeconds(10))
                    .build();

            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create("https://httpbin.org/ip"))
                    .timeout(Duration.ofSeconds(10))
                    .GET()
                    .build();

            HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());

            System.out.println("Status: " + response.statusCode());
            System.out.println("Response: " + response.body());
        } catch (Exception e) {
            System.out.println("Error: " + e.getMessage());
        }

        System.out.println();
    }

    /**
     * Example 4: Error Handling
     */
    private static void exampleErrorHandling() {
        printSeparator("Example 4: Error Handling");

        HttpClient client = HttpClient.newBuilder()
                .proxy(ProxySelector.of(new InetSocketAddress(PROXY_HOST, PROXY_PORT)))
                .connectTimeout(Duration.ofSeconds(10))
                .build();

        // Test connection timeout
        System.out.println("Testing connection timeout...");
        try {
            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create("http://httpbin.org/delay/5"))
                    .timeout(Duration.ofSeconds(2)) // Short timeout
                    .GET()
                    .build();

            client.send(request, HttpResponse.BodyHandlers.ofString());
        } catch (HttpTimeoutException e) {
            System.out.println("  Caught timeout error (expected)");
        } catch (Exception e) {
            System.out.println("  Error: " + e.getClass().getSimpleName());
        }

        // Test invalid URL
        System.out.println("Testing invalid URL...");
        try {
            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create("http://invalid.invalid.invalid"))
                    .timeout(Duration.ofSeconds(5))
                    .GET()
                    .build();

            client.send(request, HttpResponse.BodyHandlers.ofString());
        } catch (Exception e) {
            System.out.println("  Caught connection error (expected): " + e.getClass().getSimpleName());
        }

        // Test proxy auth failure
        System.out.println("Testing proxy auth failure (if auth required on proxy)...");
        try {
            // Create authenticator with wrong credentials
            Authenticator badAuthenticator = new Authenticator() {
                @Override
                protected PasswordAuthentication getPasswordAuthentication() {
                    if (getRequestorType() == RequestorType.PROXY) {
                        return new PasswordAuthentication("wronguser", "wrongpass".toCharArray());
                    }
                    return null;
                }
            };

            HttpClient badClient = HttpClient.newBuilder()
                    .proxy(ProxySelector.of(new InetSocketAddress(PROXY_HOST, PROXY_PORT)))
                    .authenticator(badAuthenticator)
                    .connectTimeout(Duration.ofSeconds(5))
                    .build();

            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create("http://httpbin.org/ip"))
                    .timeout(Duration.ofSeconds(5))
                    .GET()
                    .build();

            HttpResponse<String> response = badClient.send(request, HttpResponse.BodyHandlers.ofString());

            if (response.statusCode() == 407) {
                System.out.println("  Got 407 Proxy Authentication Required (expected)");
            } else {
                System.out.println("  Status: " + response.statusCode());
            }
        } catch (Exception e) {
            System.out.println("  Error: " + e.getMessage());
        }

        System.out.println();
    }

    /**
     * Example 5: Concurrent Requests (Load Balancing Demo)
     */
    private static void exampleConcurrentRequests() {
        printSeparator("Example 5: Concurrent Requests (Load Balancing Demo)");

        int numRequests = 10;
        System.out.printf("Making %d concurrent requests...%n", numRequests);

        ExecutorService executor = Executors.newFixedThreadPool(5);

        HttpClient client = HttpClient.newBuilder()
                .proxy(ProxySelector.of(new InetSocketAddress(PROXY_HOST, PROXY_PORT)))
                .connectTimeout(Duration.ofSeconds(10))
                .executor(executor)
                .build();

        List<CompletableFuture<String>> futures = new ArrayList<>();

        for (int i = 0; i < numRequests; i++) {
            final int requestId = i;

            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create("http://httpbin.org/ip"))
                    .timeout(Duration.ofSeconds(10))
                    .GET()
                    .build();

            CompletableFuture<String> future = client
                    .sendAsync(request, HttpResponse.BodyHandlers.ofString())
                    .thenApply(response -> {
                        String ip = extractOrigin(response.body());
                        System.out.printf("  Request %d: %s%n", requestId, ip);
                        return ip;
                    })
                    .exceptionally(e -> {
                        System.out.printf("  Request %d: error: %s%n", requestId, e.getMessage());
                        return "error";
                    });

            futures.add(future);
        }

        // Wait for all requests to complete
        CompletableFuture.allOf(futures.toArray(new CompletableFuture[0])).join();

        // Count IP distribution
        Map<String, Integer> ipCounts = new HashMap<>();
        for (CompletableFuture<String> future : futures) {
            try {
                String ip = future.get();
                if (!ip.equals("error")) {
                    ipCounts.merge(ip, 1, Integer::sum);
                }
            } catch (Exception ignored) {
            }
        }

        System.out.println("\nIP Distribution:");
        ipCounts.forEach((ip, count) -> System.out.printf("  %s: %d requests%n", ip, count));

        executor.shutdown();
        System.out.println();
    }

    /**
     * Extract the "origin" field from JSON response
     */
    private static String extractOrigin(String json) {
        // Simple extraction without JSON library
        int start = json.indexOf("\"origin\"");
        if (start == -1) return "unknown";

        int colon = json.indexOf(":", start);
        int quote1 = json.indexOf("\"", colon);
        int quote2 = json.indexOf("\"", quote1 + 1);

        if (quote1 != -1 && quote2 != -1) {
            return json.substring(quote1 + 1, quote2);
        }
        return "unknown";
    }
}
