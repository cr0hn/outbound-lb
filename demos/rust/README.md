# Rust Demo

Demonstrates how to use Outbound LB proxy with Rust.

## Requirements

- Rust 1.70+
- Dependencies defined in Cargo.toml

## Running the Demo

```bash
# Make sure the proxy is running first
cargo run
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
5. **Concurrent Requests** - Tokio async for parallel requests

## Quick Code Snippet

```rust
use reqwest::Proxy;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let client = reqwest::blocking::Client::builder()
        .proxy(Proxy::all("http://localhost:3128")?)
        .build()?;

    let response = client.get("https://api.example.com").send()?;
    println!("{}", response.text()?);

    Ok(())
}
```

## With Authentication

```rust
use reqwest::Proxy;

let client = reqwest::blocking::Client::builder()
    .proxy(Proxy::all("http://user:password@localhost:3128")?)
    .build()?;
```

## Async Example

```rust
use reqwest::Proxy;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let client = reqwest::Client::builder()
        .proxy(Proxy::all("http://localhost:3128")?)
        .build()?;

    let response = client.get("https://api.example.com").send().await?;
    println!("{}", response.text().await?);

    Ok(())
}
```

## Using Environment Variables

reqwest respects the `HTTP_PROXY` and `HTTPS_PROXY` environment variables:

```bash
export HTTP_PROXY=http://localhost:3128
export HTTPS_PROXY=http://localhost:3128
cargo run
```

```rust
// This will automatically use HTTP_PROXY/HTTPS_PROXY
let client = reqwest::blocking::Client::new();
```
