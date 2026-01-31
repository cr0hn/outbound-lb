//! Outbound LB - Rust Demo
//!
//! Demonstrates how to use Outbound LB proxy with Rust using reqwest.
//! Includes examples for HTTP, HTTPS, authentication, error handling, and concurrent requests.
//!
//! Requirements:
//!     - Rust 1.70+
//!     - Dependencies in Cargo.toml
//!
//! Usage:
//!     cargo run

use reqwest::Proxy;
use serde::Deserialize;
use std::collections::HashMap;
use std::env;
use std::time::Duration;

#[derive(Debug, Deserialize)]
struct IpResponse {
    origin: String,
}

fn get_env(key: &str, default: &str) -> String {
    env::var(key).unwrap_or_else(|_| default.to_string())
}

fn get_proxy_url() -> String {
    let host = get_env("PROXY_HOST", "localhost");
    let port = get_env("PROXY_PORT", "3128");
    format!("http://{}:{}", host, port)
}

fn get_proxy_url_with_auth() -> String {
    let host = get_env("PROXY_HOST", "localhost");
    let port = get_env("PROXY_PORT", "3128");
    let user = get_env("PROXY_USER", "user");
    let pass = get_env("PROXY_PASS", "password");
    format!("http://{}:{}@{}:{}", user, pass, host, port)
}

fn print_separator(title: &str) {
    println!("{}", "=".repeat(60));
    println!("{}", title);
    println!("{}", "=".repeat(60));
}

/// Example 1: Basic HTTP Request
fn example_http_request() {
    print_separator("Example 1: Basic HTTP Request");

    let proxy_url = get_proxy_url();

    let client = match reqwest::blocking::Client::builder()
        .proxy(Proxy::http(&proxy_url).expect("Invalid proxy URL"))
        .timeout(Duration::from_secs(10))
        .build()
    {
        Ok(c) => c,
        Err(e) => {
            println!("Error creating client: {}", e);
            println!();
            return;
        }
    };

    match client.get("http://httpbin.org/ip").send() {
        Ok(response) => {
            println!("Status: {}", response.status());
            match response.text() {
                Ok(body) => println!("Response: {}", body),
                Err(e) => println!("Error reading body: {}", e),
            }
        }
        Err(e) => println!("Error: {}", e),
    }

    println!();
}

/// Example 2: HTTPS Request (CONNECT tunnel)
fn example_https_request() {
    print_separator("Example 2: HTTPS Request (CONNECT tunnel)");

    let proxy_url = get_proxy_url();

    let client = match reqwest::blocking::Client::builder()
        .proxy(Proxy::https(&proxy_url).expect("Invalid proxy URL"))
        .timeout(Duration::from_secs(10))
        .build()
    {
        Ok(c) => c,
        Err(e) => {
            println!("Error creating client: {}", e);
            println!();
            return;
        }
    };

    match client.get("https://httpbin.org/ip").send() {
        Ok(response) => {
            println!("Status: {}", response.status());
            match response.text() {
                Ok(body) => println!("Response: {}", body),
                Err(e) => println!("Error reading body: {}", e),
            }
        }
        Err(e) => println!("Error: {}", e),
    }

    println!();
}

/// Example 3: Authenticated Proxy
fn example_authenticated_proxy() {
    print_separator("Example 3: Authenticated Proxy");

    let proxy_url = get_proxy_url_with_auth();

    let client = match reqwest::blocking::Client::builder()
        .proxy(Proxy::all(&proxy_url).expect("Invalid proxy URL"))
        .timeout(Duration::from_secs(10))
        .build()
    {
        Ok(c) => c,
        Err(e) => {
            println!("Error creating client: {}", e);
            println!();
            return;
        }
    };

    match client.get("https://httpbin.org/ip").send() {
        Ok(response) => {
            println!("Status: {}", response.status());
            match response.text() {
                Ok(body) => println!("Response: {}", body),
                Err(e) => println!("Error reading body: {}", e),
            }
        }
        Err(e) => println!("Error: {}", e),
    }

    println!();
}

/// Example 4: Error Handling
fn example_error_handling() {
    print_separator("Example 4: Error Handling");

    let proxy_url = get_proxy_url();

    // Test connection timeout
    println!("Testing connection timeout...");
    let client = reqwest::blocking::Client::builder()
        .proxy(Proxy::http(&proxy_url).expect("Invalid proxy URL"))
        .timeout(Duration::from_secs(2)) // Short timeout
        .build()
        .expect("Failed to build client");

    match client.get("http://httpbin.org/delay/5").send() {
        Ok(_) => println!("  Request succeeded unexpectedly"),
        Err(e) => {
            if e.is_timeout() {
                println!("  Caught timeout error (expected)");
            } else {
                println!("  Error: {}", e);
            }
        }
    }

    // Test invalid URL
    println!("Testing invalid URL...");
    let client = reqwest::blocking::Client::builder()
        .proxy(Proxy::http(&proxy_url).expect("Invalid proxy URL"))
        .timeout(Duration::from_secs(5))
        .build()
        .expect("Failed to build client");

    match client.get("http://invalid.invalid.invalid").send() {
        Ok(_) => println!("  Request succeeded unexpectedly"),
        Err(e) => {
            if e.is_connect() {
                println!("  Caught connection error (expected)");
            } else {
                println!("  Error: {}", e);
            }
        }
    }

    // Test proxy auth failure
    println!("Testing proxy auth failure (if auth required on proxy)...");
    let host = get_env("PROXY_HOST", "localhost");
    let port = get_env("PROXY_PORT", "3128");
    let bad_proxy_url = format!("http://wronguser:wrongpass@{}:{}", host, port);

    let client = reqwest::blocking::Client::builder()
        .proxy(Proxy::all(&bad_proxy_url).expect("Invalid proxy URL"))
        .timeout(Duration::from_secs(5))
        .build()
        .expect("Failed to build client");

    match client.get("http://httpbin.org/ip").send() {
        Ok(response) => {
            if response.status().as_u16() == 407 {
                println!("  Got 407 Proxy Authentication Required (expected)");
            } else {
                println!("  Status: {}", response.status());
            }
        }
        Err(e) => println!("  Error: {}", e),
    }

    println!();
}

/// Example 5: Concurrent Requests (Load Balancing Demo)
#[tokio::main]
async fn example_concurrent_requests() {
    print_separator("Example 5: Concurrent Requests (Load Balancing Demo)");

    let num_requests = 10;
    println!("Making {} concurrent requests...", num_requests);

    let proxy_url = get_proxy_url();

    let client = match reqwest::Client::builder()
        .proxy(Proxy::http(&proxy_url).expect("Invalid proxy URL"))
        .timeout(Duration::from_secs(10))
        .build()
    {
        Ok(c) => c,
        Err(e) => {
            println!("Error creating client: {}", e);
            println!();
            return;
        }
    };

    let mut handles = Vec::new();

    for i in 0..num_requests {
        let client = client.clone();
        let handle = tokio::spawn(async move {
            match client.get("http://httpbin.org/ip").send().await {
                Ok(response) => match response.json::<IpResponse>().await {
                    Ok(ip_resp) => (i, Ok(ip_resp.origin)),
                    Err(e) => (i, Err(e.to_string())),
                },
                Err(e) => (i, Err(e.to_string())),
            }
        });
        handles.push(handle);
    }

    let mut ip_counts: HashMap<String, i32> = HashMap::new();

    for handle in handles {
        match handle.await {
            Ok((i, Ok(ip))) => {
                println!("  Request {}: {}", i, ip);
                *ip_counts.entry(ip).or_insert(0) += 1;
            }
            Ok((i, Err(e))) => {
                println!("  Request {}: error: {}", i, e);
            }
            Err(e) => {
                println!("  Join error: {}", e);
            }
        }
    }

    println!("\nIP Distribution:");
    let mut sorted_ips: Vec<_> = ip_counts.iter().collect();
    sorted_ips.sort_by_key(|(ip, _)| ip.as_str());
    for (ip, count) in sorted_ips {
        println!("  {}: {} requests", ip, count);
    }

    println!();
}

fn main() {
    println!();
    println!("Outbound LB - Rust Demo");
    println!("Proxy: {}", get_proxy_url());
    println!();

    example_http_request();
    example_https_request();
    example_authenticated_proxy();
    example_error_handling();
    example_concurrent_requests();

    println!("All examples completed!");
}
