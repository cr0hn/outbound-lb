# Node.js Demo

Demonstrates how to use Outbound LB proxy with Node.js.

## Requirements

```bash
npm install
```

Or manually:

```bash
npm install axios https-proxy-agent http-proxy-agent
```

## Running the Demo

```bash
# Make sure the proxy is running first
node basic_proxy.js

# Or using npm
npm start
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
5. **Concurrent Requests** - Promise.all for parallel requests
6. **Native HTTP Module** - Using Node.js built-in http module

## Quick Code Snippet

```javascript
const axios = require('axios');
const { HttpsProxyAgent } = require('https-proxy-agent');

const agent = new HttpsProxyAgent('http://localhost:3128');

const response = await axios.get('https://api.example.com', {
    httpsAgent: agent,
    proxy: false,
});
console.log(response.data);
```

## With Authentication

```javascript
const { HttpsProxyAgent } = require('https-proxy-agent');

const agent = new HttpsProxyAgent('http://user:password@localhost:3128');

const response = await axios.get('https://api.example.com', {
    httpsAgent: agent,
    proxy: false,
});
```

## Using fetch (Node.js 18+)

```javascript
const { HttpsProxyAgent } = require('https-proxy-agent');

const agent = new HttpsProxyAgent('http://localhost:3128');

const response = await fetch('https://api.example.com', {
    agent,
});
const data = await response.json();
```
