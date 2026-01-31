# PHP Demo

Demonstrates how to use Outbound LB proxy with PHP.

## Requirements

- PHP 7.4+
- cURL extension enabled

## Running the Demo

```bash
# Make sure the proxy is running first
php basic_proxy.php
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
5. **Concurrent Requests** - curl_multi for parallel requests
6. **Stream Context** - Using file_get_contents with proxy

## Quick Code Snippet

```php
<?php
$ch = curl_init();

curl_setopt_array($ch, [
    CURLOPT_URL => 'https://api.example.com',
    CURLOPT_PROXY => 'localhost',
    CURLOPT_PROXYPORT => 3128,
    CURLOPT_HTTPPROXYTUNNEL => true,
    CURLOPT_RETURNTRANSFER => true,
]);

$response = curl_exec($ch);
curl_close($ch);

echo $response;
```

## With Authentication

```php
<?php
curl_setopt_array($ch, [
    CURLOPT_URL => 'https://api.example.com',
    CURLOPT_PROXY => 'localhost',
    CURLOPT_PROXYPORT => 3128,
    CURLOPT_PROXYUSERPWD => 'user:password',
    CURLOPT_HTTPPROXYTUNNEL => true,
    CURLOPT_RETURNTRANSFER => true,
]);
```

## Using Guzzle

```php
<?php
use GuzzleHttp\Client;

$client = new Client([
    'proxy' => 'http://localhost:3128',
]);

$response = $client->get('https://api.example.com');
echo $response->getBody();
```

## Using file_get_contents

```php
<?php
$context = stream_context_create([
    'http' => [
        'proxy' => 'tcp://localhost:3128',
        'request_fulluri' => true,
    ],
]);

$response = file_get_contents('http://api.example.com', false, $context);
```
