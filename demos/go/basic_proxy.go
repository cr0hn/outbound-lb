// Outbound LB - Go Demo
//
// Demonstrates how to use Outbound LB proxy with Go using net/http.
// Includes examples for HTTP, HTTPS, authentication, error handling, and concurrent requests.
//
// Usage:
//
//	go run basic_proxy.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

// Configuration from environment variables
var (
	proxyHost = getEnv("PROXY_HOST", "localhost")
	proxyPort = getEnv("PROXY_PORT", "3128")
	proxyUser = getEnv("PROXY_USER", "user")
	proxyPass = getEnv("PROXY_PASS", "password")
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getProxyURL() *url.URL {
	proxyURL, _ := url.Parse(fmt.Sprintf("http://%s:%s", proxyHost, proxyPort))
	return proxyURL
}

func getProxyURLWithAuth() *url.URL {
	proxyURL, _ := url.Parse(fmt.Sprintf("http://%s:%s@%s:%s", proxyUser, proxyPass, proxyHost, proxyPort))
	return proxyURL
}

// IPResponse represents the response from httpbin.org/ip
type IPResponse struct {
	Origin string `json:"origin"`
}

func printSeparator(title string) {
	fmt.Println(string(make([]byte, 60)))
	fmt.Printf("%s\n", "============================================================")
	fmt.Printf("%s\n", title)
	fmt.Printf("%s\n", "============================================================")
}

// Example 1: Basic HTTP Request
func exampleHTTPRequest() {
	printSeparator("Example 1: Basic HTTP Request")

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(getProxyURL()),
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get("http://httpbin.org/ip")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println()
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", body)
	fmt.Println()
}

// Example 2: HTTPS Request (CONNECT tunnel)
func exampleHTTPSRequest() {
	printSeparator("Example 2: HTTPS Request (CONNECT tunnel)")

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(getProxyURL()),
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get("https://httpbin.org/ip")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println()
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", body)
	fmt.Println()
}

// Example 3: Authenticated Proxy
func exampleAuthenticatedProxy() {
	printSeparator("Example 3: Authenticated Proxy")

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(getProxyURLWithAuth()),
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get("https://httpbin.org/ip")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println()
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", body)
	fmt.Println()
}

// Example 4: Error Handling
func exampleErrorHandling() {
	printSeparator("Example 4: Error Handling")

	// Test connection timeout
	fmt.Println("Testing connection timeout...")
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(getProxyURL()),
		},
		Timeout: 2 * time.Second,
	}

	_, err := client.Get("http://httpbin.org/delay/5")
	if err != nil {
		fmt.Println("  Caught timeout error (expected)")
	}

	// Test invalid URL
	fmt.Println("Testing invalid URL...")
	client2 := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(getProxyURL()),
		},
		Timeout: 5 * time.Second,
	}

	_, err = client2.Get("http://invalid.invalid.invalid")
	if err != nil {
		fmt.Printf("  Caught connection error (expected): %T\n", err)
	}

	// Test proxy authentication failure
	fmt.Println("Testing proxy auth failure (if auth required on proxy)...")
	badProxyURL, _ := url.Parse(fmt.Sprintf("http://wronguser:wrongpass@%s:%s", proxyHost, proxyPort))
	client3 := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(badProxyURL),
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client3.Get("http://httpbin.org/ip")
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode == 407 {
			fmt.Println("  Got 407 Proxy Authentication Required (expected)")
		} else {
			fmt.Printf("  Status: %d\n", resp.StatusCode)
		}
	}

	fmt.Println()
}

// Example 5: Concurrent Requests (Load Balancing Demo)
func exampleConcurrentRequests() {
	printSeparator("Example 5: Concurrent Requests (Load Balancing Demo)")

	numRequests := 10
	fmt.Printf("Making %d concurrent requests...\n", numRequests)

	type result struct {
		requestID int
		ip        string
		err       error
	}

	results := make(chan result, numRequests)
	var wg sync.WaitGroup

	client := &http.Client{
		Transport: &http.Transport{
			Proxy:               http.ProxyURL(getProxyURL()),
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
		},
		Timeout: 10 * time.Second,
	}

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			resp, err := client.Get("http://httpbin.org/ip")
			if err != nil {
				results <- result{requestID: id, err: err}
				return
			}
			defer resp.Body.Close()

			var ipResp IPResponse
			if err := json.NewDecoder(resp.Body).Decode(&ipResp); err != nil {
				results <- result{requestID: id, err: err}
				return
			}

			results <- result{requestID: id, ip: ipResp.Origin}
		}(i)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	ipCounts := make(map[string]int)
	for r := range results {
		if r.err != nil {
			fmt.Printf("  Request %d: error: %v\n", r.requestID, r.err)
		} else {
			fmt.Printf("  Request %d: %s\n", r.requestID, r.ip)
			ipCounts[r.ip]++
		}
	}

	fmt.Println("\nIP Distribution:")
	for ip, count := range ipCounts {
		fmt.Printf("  %s: %d requests\n", ip, count)
	}

	fmt.Println()
}

// Example 6: Context with Cancellation
func exampleContextCancellation() {
	printSeparator("Example 6: Context with Cancellation")

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(getProxyURL()),
		},
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://httpbin.org/ip", nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		fmt.Println()
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println()
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", body)
	fmt.Println()
}

func main() {
	fmt.Println()
	fmt.Println("Outbound LB - Go Demo")
	fmt.Printf("Proxy: http://%s:%s\n", proxyHost, proxyPort)
	fmt.Println()

	exampleHTTPRequest()
	exampleHTTPSRequest()
	exampleAuthenticatedProxy()
	exampleErrorHandling()
	exampleConcurrentRequests()
	exampleContextCancellation()

	fmt.Println("All examples completed!")
}
