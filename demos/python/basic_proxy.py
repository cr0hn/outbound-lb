#!/usr/bin/env python3
"""
Outbound LB - Python Demo

Demonstrates how to use Outbound LB proxy with Python using the requests library.
Includes examples for HTTP, HTTPS, authentication, error handling, and concurrent requests.

Requirements:
    pip install requests aiohttp

Usage:
    python basic_proxy.py
"""

import os
import sys
import asyncio
from concurrent.futures import ThreadPoolExecutor, as_completed

# Configuration from environment variables
PROXY_HOST = os.getenv("PROXY_HOST", "localhost")
PROXY_PORT = os.getenv("PROXY_PORT", "3128")
PROXY_USER = os.getenv("PROXY_USER", "user")
PROXY_PASS = os.getenv("PROXY_PASS", "password")

# Proxy URLs
PROXY_URL = f"http://{PROXY_HOST}:{PROXY_PORT}"
PROXY_URL_AUTH = f"http://{PROXY_USER}:{PROXY_PASS}@{PROXY_HOST}:{PROXY_PORT}"


def example_http_request():
    """Basic HTTP request through the proxy."""
    import requests

    print("=" * 60)
    print("Example 1: Basic HTTP Request")
    print("=" * 60)

    proxies = {
        "http": PROXY_URL,
        "https": PROXY_URL,
    }

    try:
        response = requests.get("http://httpbin.org/ip", proxies=proxies, timeout=10)
        print(f"Status: {response.status_code}")
        print(f"Response: {response.json()}")
    except requests.RequestException as e:
        print(f"Error: {e}")

    print()


def example_https_request():
    """HTTPS request through the proxy (CONNECT tunnel)."""
    import requests

    print("=" * 60)
    print("Example 2: HTTPS Request (CONNECT tunnel)")
    print("=" * 60)

    proxies = {
        "http": PROXY_URL,
        "https": PROXY_URL,
    }

    try:
        response = requests.get("https://httpbin.org/ip", proxies=proxies, timeout=10)
        print(f"Status: {response.status_code}")
        print(f"Response: {response.json()}")
    except requests.RequestException as e:
        print(f"Error: {e}")

    print()


def example_authenticated_proxy():
    """Request through authenticated proxy."""
    import requests

    print("=" * 60)
    print("Example 3: Authenticated Proxy")
    print("=" * 60)

    proxies = {
        "http": PROXY_URL_AUTH,
        "https": PROXY_URL_AUTH,
    }

    try:
        response = requests.get("https://httpbin.org/ip", proxies=proxies, timeout=10)
        print(f"Status: {response.status_code}")
        print(f"Response: {response.json()}")
    except requests.RequestException as e:
        print(f"Error: {e}")

    print()


def example_error_handling():
    """Demonstrate error handling for common scenarios."""
    import requests

    print("=" * 60)
    print("Example 4: Error Handling")
    print("=" * 60)

    proxies = {
        "http": PROXY_URL,
        "https": PROXY_URL,
    }

    # Test connection timeout
    print("Testing connection timeout...")
    try:
        response = requests.get(
            "http://httpbin.org/delay/5",
            proxies=proxies,
            timeout=2,  # Short timeout to trigger error
        )
    except requests.Timeout:
        print("  Caught timeout error (expected)")
    except requests.RequestException as e:
        print(f"  Error: {e}")

    # Test invalid URL
    print("Testing invalid URL...")
    try:
        response = requests.get(
            "http://invalid.invalid.invalid",
            proxies=proxies,
            timeout=5,
        )
    except requests.RequestException as e:
        print(f"  Caught connection error (expected): {type(e).__name__}")

    # Test proxy authentication failure (if auth is required)
    print("Testing proxy auth failure (if auth required on proxy)...")
    try:
        bad_proxy = f"http://wronguser:wrongpass@{PROXY_HOST}:{PROXY_PORT}"
        response = requests.get(
            "http://httpbin.org/ip",
            proxies={"http": bad_proxy, "https": bad_proxy},
            timeout=5,
        )
        if response.status_code == 407:
            print("  Got 407 Proxy Authentication Required (expected)")
        else:
            print(f"  Status: {response.status_code}")
    except requests.RequestException as e:
        print(f"  Error: {e}")

    print()


def example_concurrent_requests():
    """Demonstrate concurrent requests to show load balancing."""
    import requests

    print("=" * 60)
    print("Example 5: Concurrent Requests (Load Balancing Demo)")
    print("=" * 60)

    proxies = {
        "http": PROXY_URL,
        "https": PROXY_URL,
    }

    def make_request(request_id):
        """Make a single request and return the source IP."""
        try:
            response = requests.get(
                "http://httpbin.org/ip",
                proxies=proxies,
                timeout=10,
            )
            return request_id, response.json().get("origin", "unknown")
        except requests.RequestException as e:
            return request_id, f"error: {e}"

    num_requests = 10
    print(f"Making {num_requests} concurrent requests...")

    # Track IP distribution
    ip_counts = {}

    with ThreadPoolExecutor(max_workers=5) as executor:
        futures = [executor.submit(make_request, i) for i in range(num_requests)]

        for future in as_completed(futures):
            request_id, ip = future.result()
            print(f"  Request {request_id}: {ip}")

            if not ip.startswith("error"):
                ip_counts[ip] = ip_counts.get(ip, 0) + 1

    print("\nIP Distribution:")
    for ip, count in sorted(ip_counts.items()):
        print(f"  {ip}: {count} requests")

    print()


async def example_async_requests():
    """Demonstrate async requests using aiohttp."""
    try:
        import aiohttp
    except ImportError:
        print("=" * 60)
        print("Example 6: Async Requests (aiohttp)")
        print("=" * 60)
        print("Skipped: aiohttp not installed")
        print("Install with: pip install aiohttp")
        print()
        return

    print("=" * 60)
    print("Example 6: Async Requests (aiohttp)")
    print("=" * 60)

    async def fetch(session, url, request_id):
        """Make an async request."""
        try:
            async with session.get(url, timeout=aiohttp.ClientTimeout(total=10)) as response:
                data = await response.json()
                return request_id, data.get("origin", "unknown")
        except Exception as e:
            return request_id, f"error: {e}"

    num_requests = 10
    print(f"Making {num_requests} async requests...")

    connector = aiohttp.TCPConnector(limit=10)

    async with aiohttp.ClientSession(
        connector=connector,
        trust_env=False,  # Don't use system proxy settings
    ) as session:
        # Set proxy for aiohttp
        session._connector._proxy = PROXY_URL

        tasks = [
            fetch(session, "http://httpbin.org/ip", i) for i in range(num_requests)
        ]

        results = await asyncio.gather(*tasks)

        ip_counts = {}
        for request_id, ip in results:
            print(f"  Request {request_id}: {ip}")
            if not ip.startswith("error"):
                ip_counts[ip] = ip_counts.get(ip, 0) + 1

        print("\nIP Distribution:")
        for ip, count in sorted(ip_counts.items()):
            print(f"  {ip}: {count} requests")

    print()


def main():
    """Run all examples."""
    print()
    print("Outbound LB - Python Demo")
    print(f"Proxy: {PROXY_URL}")
    print()

    # Check if requests is available
    try:
        import requests
    except ImportError:
        print("Error: 'requests' library is required")
        print("Install with: pip install requests")
        sys.exit(1)

    # Run examples
    example_http_request()
    example_https_request()
    example_authenticated_proxy()
    example_error_handling()
    example_concurrent_requests()

    # Run async example
    asyncio.run(example_async_requests())

    print("All examples completed!")


if __name__ == "__main__":
    main()
