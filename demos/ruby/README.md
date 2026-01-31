# Ruby Demo

Demonstrates how to use Outbound LB proxy with Ruby.

## Requirements

- Ruby 2.7+

## Running the Demo

```bash
# Make sure the proxy is running first
ruby basic_proxy.rb
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
5. **Concurrent Requests** - Threads for parallel requests
6. **Environment Variables** - Using HTTP_PROXY/HTTPS_PROXY

## Quick Code Snippet

```ruby
require 'net/http'
require 'uri'

uri = URI('https://api.example.com')

http = Net::HTTP.new(uri.host, uri.port, 'localhost', 3128)
http.use_ssl = true

response = http.get(uri.request_uri)
puts response.body
```

## With Authentication

```ruby
http = Net::HTTP.new(uri.host, uri.port, 'localhost', 3128, 'user', 'password')
http.use_ssl = true

response = http.get(uri.request_uri)
```

## Using Environment Variables

Ruby's Net::HTTP respects the `HTTP_PROXY` and `HTTPS_PROXY` environment variables:

```bash
export HTTP_PROXY=http://localhost:3128
export HTTPS_PROXY=http://localhost:3128
ruby yourscript.rb
```

```ruby
# This will automatically use HTTP_PROXY/HTTPS_PROXY
require 'net/http'
require 'uri'

uri = URI('https://api.example.com')
response = Net::HTTP.get_response(uri)
```

## Using with Faraday

```ruby
require 'faraday'

conn = Faraday.new(proxy: 'http://localhost:3128') do |f|
  f.adapter Faraday.default_adapter
end

response = conn.get('https://api.example.com')
```
