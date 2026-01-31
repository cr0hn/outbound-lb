# Python Demo

Demonstrates how to use Outbound LB proxy with Python.

## Requirements

```bash
pip install -r requirements.txt
```

Or manually:

```bash
pip install requests aiohttp
```

## Running the Demo

```bash
# Make sure the proxy is running first
python basic_proxy.py
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
5. **Concurrent Requests** - ThreadPoolExecutor for parallel requests
6. **Async Requests** - Using aiohttp for async/await pattern

## Quick Code Snippet

```python
import requests

proxies = {
    'http': 'http://localhost:3128',
    'https': 'http://localhost:3128',
}

response = requests.get('https://api.example.com', proxies=proxies)
print(response.json())
```

## With Authentication

```python
import requests

proxies = {
    'http': 'http://user:password@localhost:3128',
    'https': 'http://user:password@localhost:3128',
}

response = requests.get('https://api.example.com', proxies=proxies)
```

## Using aiohttp

```python
import aiohttp

async with aiohttp.ClientSession() as session:
    async with session.get(
        'https://api.example.com',
        proxy='http://localhost:3128'
    ) as response:
        data = await response.json()
```
