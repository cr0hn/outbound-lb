#!/usr/bin/env php
<?php
/**
 * Outbound LB - PHP Demo
 *
 * Demonstrates how to use Outbound LB proxy with PHP using cURL.
 * Includes examples for HTTP, HTTPS, authentication, error handling, and concurrent requests.
 *
 * Requirements:
 *     PHP 7.4+ with cURL extension
 *
 * Usage:
 *     php basic_proxy.php
 */

// Configuration from environment variables
$PROXY_HOST = getenv('PROXY_HOST') ?: 'localhost';
$PROXY_PORT = getenv('PROXY_PORT') ?: '3128';
$PROXY_USER = getenv('PROXY_USER') ?: 'user';
$PROXY_PASS = getenv('PROXY_PASS') ?: 'password';

$PROXY_URL = "http://{$PROXY_HOST}:{$PROXY_PORT}";

function printSeparator(string $title): void {
    echo str_repeat('=', 60) . "\n";
    echo $title . "\n";
    echo str_repeat('=', 60) . "\n";
}

/**
 * Example 1: Basic HTTP Request
 */
function exampleHttpRequest(): void {
    global $PROXY_HOST, $PROXY_PORT;

    printSeparator('Example 1: Basic HTTP Request');

    $ch = curl_init();

    curl_setopt_array($ch, [
        CURLOPT_URL => 'http://httpbin.org/ip',
        CURLOPT_PROXY => $PROXY_HOST,
        CURLOPT_PROXYPORT => $PROXY_PORT,
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_TIMEOUT => 10,
        CURLOPT_CONNECTTIMEOUT => 10,
    ]);

    $response = curl_exec($ch);
    $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
    $error = curl_error($ch);

    curl_close($ch);

    if ($error) {
        echo "Error: {$error}\n";
    } else {
        echo "Status: {$httpCode}\n";
        echo "Response: {$response}\n";
    }

    echo "\n";
}

/**
 * Example 2: HTTPS Request (CONNECT tunnel)
 */
function exampleHttpsRequest(): void {
    global $PROXY_HOST, $PROXY_PORT;

    printSeparator('Example 2: HTTPS Request (CONNECT tunnel)');

    $ch = curl_init();

    curl_setopt_array($ch, [
        CURLOPT_URL => 'https://httpbin.org/ip',
        CURLOPT_PROXY => $PROXY_HOST,
        CURLOPT_PROXYPORT => $PROXY_PORT,
        CURLOPT_HTTPPROXYTUNNEL => true, // Use CONNECT for HTTPS
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_TIMEOUT => 10,
        CURLOPT_CONNECTTIMEOUT => 10,
        CURLOPT_SSL_VERIFYPEER => true,
    ]);

    $response = curl_exec($ch);
    $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
    $error = curl_error($ch);

    curl_close($ch);

    if ($error) {
        echo "Error: {$error}\n";
    } else {
        echo "Status: {$httpCode}\n";
        echo "Response: {$response}\n";
    }

    echo "\n";
}

/**
 * Example 3: Authenticated Proxy
 */
function exampleAuthenticatedProxy(): void {
    global $PROXY_HOST, $PROXY_PORT, $PROXY_USER, $PROXY_PASS;

    printSeparator('Example 3: Authenticated Proxy');

    $ch = curl_init();

    curl_setopt_array($ch, [
        CURLOPT_URL => 'https://httpbin.org/ip',
        CURLOPT_PROXY => $PROXY_HOST,
        CURLOPT_PROXYPORT => $PROXY_PORT,
        CURLOPT_PROXYUSERPWD => "{$PROXY_USER}:{$PROXY_PASS}",
        CURLOPT_HTTPPROXYTUNNEL => true,
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_TIMEOUT => 10,
        CURLOPT_CONNECTTIMEOUT => 10,
        CURLOPT_SSL_VERIFYPEER => true,
    ]);

    $response = curl_exec($ch);
    $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
    $error = curl_error($ch);

    curl_close($ch);

    if ($error) {
        echo "Error: {$error}\n";
    } else {
        echo "Status: {$httpCode}\n";
        echo "Response: {$response}\n";
    }

    echo "\n";
}

/**
 * Example 4: Error Handling
 */
function exampleErrorHandling(): void {
    global $PROXY_HOST, $PROXY_PORT;

    printSeparator('Example 4: Error Handling');

    // Test connection timeout
    echo "Testing connection timeout...\n";

    $ch = curl_init();
    curl_setopt_array($ch, [
        CURLOPT_URL => 'http://httpbin.org/delay/5',
        CURLOPT_PROXY => $PROXY_HOST,
        CURLOPT_PROXYPORT => $PROXY_PORT,
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_TIMEOUT => 2, // Short timeout
    ]);

    $response = curl_exec($ch);
    $error = curl_error($ch);
    $errno = curl_errno($ch);
    curl_close($ch);

    if ($errno === CURLE_OPERATION_TIMEOUTED) {
        echo "  Caught timeout error (expected)\n";
    } else if ($error) {
        echo "  Error: {$error}\n";
    }

    // Test invalid URL
    echo "Testing invalid URL...\n";

    $ch = curl_init();
    curl_setopt_array($ch, [
        CURLOPT_URL => 'http://invalid.invalid.invalid',
        CURLOPT_PROXY => $PROXY_HOST,
        CURLOPT_PROXYPORT => $PROXY_PORT,
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_TIMEOUT => 5,
    ]);

    $response = curl_exec($ch);
    $error = curl_error($ch);
    $errno = curl_errno($ch);
    curl_close($ch);

    if ($error) {
        echo "  Caught connection error (expected): CURLE_{$errno}\n";
    }

    // Test proxy auth failure
    echo "Testing proxy auth failure (if auth required on proxy)...\n";

    $ch = curl_init();
    curl_setopt_array($ch, [
        CURLOPT_URL => 'http://httpbin.org/ip',
        CURLOPT_PROXY => $PROXY_HOST,
        CURLOPT_PROXYPORT => $PROXY_PORT,
        CURLOPT_PROXYUSERPWD => 'wronguser:wrongpass',
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_TIMEOUT => 5,
    ]);

    $response = curl_exec($ch);
    $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
    $error = curl_error($ch);
    curl_close($ch);

    if ($httpCode === 407) {
        echo "  Got 407 Proxy Authentication Required (expected)\n";
    } else if ($error) {
        echo "  Error: {$error}\n";
    } else {
        echo "  Status: {$httpCode}\n";
    }

    echo "\n";
}

/**
 * Example 5: Concurrent Requests (Load Balancing Demo)
 */
function exampleConcurrentRequests(): void {
    global $PROXY_HOST, $PROXY_PORT;

    printSeparator('Example 5: Concurrent Requests (Load Balancing Demo)');

    $numRequests = 10;
    echo "Making {$numRequests} concurrent requests...\n";

    $multiHandle = curl_multi_init();
    $handles = [];

    // Create handles
    for ($i = 0; $i < $numRequests; $i++) {
        $ch = curl_init();
        curl_setopt_array($ch, [
            CURLOPT_URL => 'http://httpbin.org/ip',
            CURLOPT_PROXY => $PROXY_HOST,
            CURLOPT_PROXYPORT => $PROXY_PORT,
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_TIMEOUT => 10,
        ]);

        curl_multi_add_handle($multiHandle, $ch);
        $handles[$i] = $ch;
    }

    // Execute all requests
    $running = null;
    do {
        curl_multi_exec($multiHandle, $running);
        curl_multi_select($multiHandle);
    } while ($running > 0);

    // Collect results
    $ipCounts = [];

    foreach ($handles as $i => $ch) {
        $response = curl_multi_getcontent($ch);
        $error = curl_error($ch);

        if ($error) {
            echo "  Request {$i}: error: {$error}\n";
        } else {
            $data = json_decode($response, true);
            $ip = $data['origin'] ?? 'unknown';
            echo "  Request {$i}: {$ip}\n";

            if (!isset($ipCounts[$ip])) {
                $ipCounts[$ip] = 0;
            }
            $ipCounts[$ip]++;
        }

        curl_multi_remove_handle($multiHandle, $ch);
        curl_close($ch);
    }

    curl_multi_close($multiHandle);

    echo "\nIP Distribution:\n";
    ksort($ipCounts);
    foreach ($ipCounts as $ip => $count) {
        echo "  {$ip}: {$count} requests\n";
    }

    echo "\n";
}

/**
 * Example 6: Using file_get_contents with stream context
 */
function exampleStreamContext(): void {
    global $PROXY_HOST, $PROXY_PORT;

    printSeparator('Example 6: Using Stream Context');

    $context = stream_context_create([
        'http' => [
            'proxy' => "tcp://{$PROXY_HOST}:{$PROXY_PORT}",
            'request_fulluri' => true,
            'timeout' => 10,
        ],
    ]);

    $response = @file_get_contents('http://httpbin.org/ip', false, $context);

    if ($response === false) {
        echo "Error: Failed to fetch URL\n";
    } else {
        echo "Response: {$response}\n";
    }

    echo "\n";
}

// Main
function main(): void {
    global $PROXY_URL;

    echo "\n";
    echo "Outbound LB - PHP Demo\n";
    echo "Proxy: {$PROXY_URL}\n";
    echo "\n";

    // Check if cURL is available
    if (!function_exists('curl_init')) {
        echo "Error: cURL extension is required\n";
        exit(1);
    }

    exampleHttpRequest();
    exampleHttpsRequest();
    exampleAuthenticatedProxy();
    exampleErrorHandling();
    exampleConcurrentRequests();
    exampleStreamContext();

    echo "All examples completed!\n";
}

main();
