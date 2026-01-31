#!/bin/bash
#
# Outbound LB - curl/bash Demo
#
# Demonstrates how to use Outbound LB proxy with curl and bash.
# Includes examples for HTTP, HTTPS, authentication, error handling, and concurrent requests.
#
# Requirements:
#     - curl
#     - bash 4+
#
# Usage:
#     chmod +x examples.sh
#     ./examples.sh
#

# Configuration from environment variables
PROXY_HOST="${PROXY_HOST:-localhost}"
PROXY_PORT="${PROXY_PORT:-3128}"
PROXY_USER="${PROXY_USER:-user}"
PROXY_PASS="${PROXY_PASS:-password}"

PROXY_URL="http://${PROXY_HOST}:${PROXY_PORT}"
PROXY_URL_AUTH="http://${PROXY_USER}:${PROXY_PASS}@${PROXY_HOST}:${PROXY_PORT}"

print_separator() {
    echo "============================================================"
    echo "$1"
    echo "============================================================"
}

# Example 1: Basic HTTP Request
example_http_request() {
    print_separator "Example 1: Basic HTTP Request"

    echo "Command: curl -x ${PROXY_URL} http://httpbin.org/ip"
    echo
    curl -s -x "${PROXY_URL}" http://httpbin.org/ip
    echo
    echo
}

# Example 2: HTTPS Request (CONNECT tunnel)
example_https_request() {
    print_separator "Example 2: HTTPS Request (CONNECT tunnel)"

    echo "Command: curl -x ${PROXY_URL} https://httpbin.org/ip"
    echo
    curl -s -x "${PROXY_URL}" https://httpbin.org/ip
    echo
    echo
}

# Example 3: Authenticated Proxy
example_authenticated_proxy() {
    print_separator "Example 3: Authenticated Proxy"

    echo "Command: curl -x ${PROXY_URL} -U ${PROXY_USER}:${PROXY_PASS} https://httpbin.org/ip"
    echo "(or: curl -x ${PROXY_URL_AUTH} https://httpbin.org/ip)"
    echo
    curl -s -x "${PROXY_URL}" -U "${PROXY_USER}:${PROXY_PASS}" https://httpbin.org/ip
    echo
    echo
}

# Example 4: Verbose output (see CONNECT)
example_verbose_output() {
    print_separator "Example 4: Verbose Output (see CONNECT handshake)"

    echo "Command: curl -v -x ${PROXY_URL} https://httpbin.org/ip 2>&1 | head -30"
    echo
    curl -v -x "${PROXY_URL}" https://httpbin.org/ip 2>&1 | head -30
    echo "..."
    echo
}

# Example 5: Error Handling
example_error_handling() {
    print_separator "Example 5: Error Handling"

    # Test connection timeout
    echo "Testing connection timeout..."
    echo "Command: curl -x ${PROXY_URL} --connect-timeout 2 http://httpbin.org/delay/5"
    result=$(curl -s -w "%{http_code}" -x "${PROXY_URL}" --connect-timeout 2 -m 2 http://httpbin.org/delay/5 2>&1)
    if [[ $? -ne 0 ]]; then
        echo "  Caught timeout error (expected)"
    else
        echo "  Result: $result"
    fi
    echo

    # Test invalid URL
    echo "Testing invalid URL..."
    echo "Command: curl -x ${PROXY_URL} http://invalid.invalid.invalid"
    result=$(curl -s -w "%{http_code}" -x "${PROXY_URL}" --connect-timeout 5 http://invalid.invalid.invalid 2>&1)
    if [[ $? -ne 0 ]]; then
        echo "  Caught connection error (expected)"
    else
        echo "  Result: $result"
    fi
    echo

    # Test proxy auth failure
    echo "Testing proxy auth failure (if auth required on proxy)..."
    echo "Command: curl -x ${PROXY_URL} -U wronguser:wrongpass http://httpbin.org/ip"
    http_code=$(curl -s -w "%{http_code}" -o /dev/null -x "${PROXY_URL}" -U "wronguser:wrongpass" http://httpbin.org/ip)
    if [[ "$http_code" == "407" ]]; then
        echo "  Got 407 Proxy Authentication Required (expected)"
    else
        echo "  HTTP Status: $http_code"
    fi
    echo
}

# Example 6: Concurrent Requests (Load Balancing Demo)
example_concurrent_requests() {
    print_separator "Example 6: Concurrent Requests (Load Balancing Demo)"

    NUM_REQUESTS=10
    echo "Making ${NUM_REQUESTS} concurrent requests..."
    echo

    # Create temp file for results
    RESULTS_FILE=$(mktemp)

    # Launch concurrent requests
    for i in $(seq 1 $NUM_REQUESTS); do
        (
            ip=$(curl -s -x "${PROXY_URL}" http://httpbin.org/ip | grep -o '"origin": "[^"]*"' | cut -d'"' -f4)
            echo "$i:$ip" >> "$RESULTS_FILE"
        ) &
    done

    # Wait for all requests to complete
    wait

    # Display results
    sort -t: -k1 -n "$RESULTS_FILE" | while IFS=: read -r num ip; do
        echo "  Request $num: $ip"
    done
    echo

    # Count IP distribution
    echo "IP Distribution:"
    cut -d: -f2 "$RESULTS_FILE" | sort | uniq -c | while read count ip; do
        echo "  $ip: $count requests"
    done

    rm -f "$RESULTS_FILE"
    echo
}

# Example 7: Using wget
example_wget() {
    print_separator "Example 7: Using wget"

    echo "Command: wget -e http_proxy=${PROXY_URL} -q -O - http://httpbin.org/ip"
    echo

    if command -v wget &> /dev/null; then
        wget -e "http_proxy=${PROXY_URL}" -e "https_proxy=${PROXY_URL}" -q -O - http://httpbin.org/ip
        echo
    else
        echo "wget not installed, skipping..."
    fi
    echo
}

# Example 8: Using HTTPie (if installed)
example_httpie() {
    print_separator "Example 8: Using HTTPie"

    echo "Command: http --proxy http:${PROXY_URL} httpbin.org/ip"
    echo

    if command -v http &> /dev/null; then
        http --proxy "http:${PROXY_URL}" httpbin.org/ip
    else
        echo "HTTPie not installed, skipping..."
        echo "Install with: pip install httpie"
    fi
    echo
}

# Example 9: POST request
example_post_request() {
    print_separator "Example 9: POST Request"

    echo "Command: curl -x ${PROXY_URL} -X POST -d '{\"key\":\"value\"}' -H 'Content-Type: application/json' http://httpbin.org/post"
    echo
    curl -s -x "${PROXY_URL}" -X POST -d '{"key":"value"}' -H 'Content-Type: application/json' http://httpbin.org/post | head -20
    echo "..."
    echo
}

# Example 10: Headers and custom options
example_headers() {
    print_separator "Example 10: Custom Headers"

    echo "Command: curl -x ${PROXY_URL} -H 'X-Custom-Header: test' http://httpbin.org/headers"
    echo
    curl -s -x "${PROXY_URL}" -H 'X-Custom-Header: test' http://httpbin.org/headers
    echo
}

# Main
main() {
    echo
    echo "Outbound LB - curl/bash Demo"
    echo "Proxy: ${PROXY_URL}"
    echo

    example_http_request
    example_https_request
    example_authenticated_proxy
    example_verbose_output
    example_error_handling
    example_concurrent_requests
    example_wget
    example_httpie
    example_post_request
    example_headers

    echo "All examples completed!"
}

main
