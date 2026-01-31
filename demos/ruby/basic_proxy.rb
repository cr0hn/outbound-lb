#!/usr/bin/env ruby
# frozen_string_literal: true

#
# Outbound LB - Ruby Demo
#
# Demonstrates how to use Outbound LB proxy with Ruby using net/http.
# Includes examples for HTTP, HTTPS, authentication, error handling, and concurrent requests.
#
# Requirements:
#     Ruby 2.7+
#
# Usage:
#     ruby basic_proxy.rb
#

require 'net/http'
require 'uri'
require 'json'
require 'timeout'

# Configuration from environment variables
PROXY_HOST = ENV.fetch('PROXY_HOST', 'localhost')
PROXY_PORT = ENV.fetch('PROXY_PORT', '3128').to_i
PROXY_USER = ENV.fetch('PROXY_USER', 'user')
PROXY_PASS = ENV.fetch('PROXY_PASS', 'password')

def print_separator(title)
  puts '=' * 60
  puts title
  puts '=' * 60
end

# Example 1: Basic HTTP Request
def example_http_request
  print_separator('Example 1: Basic HTTP Request')

  begin
    uri = URI('http://httpbin.org/ip')

    http = Net::HTTP.new(uri.host, uri.port, PROXY_HOST, PROXY_PORT)
    http.open_timeout = 10
    http.read_timeout = 10

    response = http.get(uri.request_uri)

    puts "Status: #{response.code}"
    puts "Response: #{response.body}"
  rescue StandardError => e
    puts "Error: #{e.message}"
  end

  puts
end

# Example 2: HTTPS Request (CONNECT tunnel)
def example_https_request
  print_separator('Example 2: HTTPS Request (CONNECT tunnel)')

  begin
    uri = URI('https://httpbin.org/ip')

    http = Net::HTTP.new(uri.host, uri.port, PROXY_HOST, PROXY_PORT)
    http.use_ssl = true
    http.verify_mode = OpenSSL::SSL::VERIFY_PEER
    http.open_timeout = 10
    http.read_timeout = 10

    response = http.get(uri.request_uri)

    puts "Status: #{response.code}"
    puts "Response: #{response.body}"
  rescue StandardError => e
    puts "Error: #{e.message}"
  end

  puts
end

# Example 3: Authenticated Proxy
def example_authenticated_proxy
  print_separator('Example 3: Authenticated Proxy')

  begin
    uri = URI('https://httpbin.org/ip')

    http = Net::HTTP.new(uri.host, uri.port, PROXY_HOST, PROXY_PORT, PROXY_USER, PROXY_PASS)
    http.use_ssl = true
    http.verify_mode = OpenSSL::SSL::VERIFY_PEER
    http.open_timeout = 10
    http.read_timeout = 10

    response = http.get(uri.request_uri)

    puts "Status: #{response.code}"
    puts "Response: #{response.body}"
  rescue StandardError => e
    puts "Error: #{e.message}"
  end

  puts
end

# Example 4: Error Handling
def example_error_handling
  print_separator('Example 4: Error Handling')

  # Test connection timeout
  puts 'Testing connection timeout...'
  begin
    uri = URI('http://httpbin.org/delay/5')

    http = Net::HTTP.new(uri.host, uri.port, PROXY_HOST, PROXY_PORT)
    http.open_timeout = 2 # Short timeout
    http.read_timeout = 2

    http.get(uri.request_uri)
  rescue Net::OpenTimeout, Net::ReadTimeout, Timeout::Error
    puts '  Caught timeout error (expected)'
  rescue StandardError => e
    puts "  Error: #{e.class.name}"
  end

  # Test invalid URL
  puts 'Testing invalid URL...'
  begin
    uri = URI('http://invalid.invalid.invalid')

    http = Net::HTTP.new(uri.host, uri.port, PROXY_HOST, PROXY_PORT)
    http.open_timeout = 5
    http.read_timeout = 5

    http.get(uri.request_uri)
  rescue SocketError, Errno::ECONNREFUSED => e
    puts "  Caught connection error (expected): #{e.class.name}"
  rescue StandardError => e
    puts "  Error: #{e.class.name}"
  end

  # Test proxy auth failure
  puts 'Testing proxy auth failure (if auth required on proxy)...'
  begin
    uri = URI('http://httpbin.org/ip')

    http = Net::HTTP.new(uri.host, uri.port, PROXY_HOST, PROXY_PORT, 'wronguser', 'wrongpass')
    http.open_timeout = 5
    http.read_timeout = 5

    response = http.get(uri.request_uri)

    if response.code == '407'
      puts '  Got 407 Proxy Authentication Required (expected)'
    else
      puts "  Status: #{response.code}"
    end
  rescue StandardError => e
    puts "  Error: #{e.message}"
  end

  puts
end

# Example 5: Concurrent Requests (Load Balancing Demo)
def example_concurrent_requests
  print_separator('Example 5: Concurrent Requests (Load Balancing Demo)')

  num_requests = 10
  puts "Making #{num_requests} concurrent requests..."

  threads = []
  results = []
  mutex = Mutex.new

  num_requests.times do |i|
    threads << Thread.new(i) do |request_id|
      begin
        uri = URI('http://httpbin.org/ip')

        http = Net::HTTP.new(uri.host, uri.port, PROXY_HOST, PROXY_PORT)
        http.open_timeout = 10
        http.read_timeout = 10

        response = http.get(uri.request_uri)
        data = JSON.parse(response.body)
        ip = data['origin']

        mutex.synchronize do
          results << { request_id: request_id, ip: ip }
        end
      rescue StandardError => e
        mutex.synchronize do
          results << { request_id: request_id, ip: "error: #{e.message}" }
        end
      end
    end
  end

  threads.each(&:join)

  # Print results
  results.sort_by { |r| r[:request_id] }.each do |r|
    puts "  Request #{r[:request_id]}: #{r[:ip]}"
  end

  # Count IP distribution
  ip_counts = {}
  results.each do |r|
    next if r[:ip].start_with?('error')

    ip_counts[r[:ip]] ||= 0
    ip_counts[r[:ip]] += 1
  end

  puts "\nIP Distribution:"
  ip_counts.sort.each do |ip, count|
    puts "  #{ip}: #{count} requests"
  end

  puts
end

# Example 6: Using environment variables
def example_environment_variables
  print_separator('Example 6: Using Environment Variables')

  puts 'Ruby respects HTTP_PROXY and HTTPS_PROXY environment variables.'
  puts 'Set them before running your script:'
  puts
  puts '  export HTTP_PROXY=http://localhost:3128'
  puts '  export HTTPS_PROXY=http://localhost:3128'
  puts
  puts 'Then use Net::HTTP without explicit proxy configuration:'
  puts
  puts '  uri = URI("https://api.example.com")'
  puts '  response = Net::HTTP.get_response(uri)'
  puts
end

# Main
def main
  puts
  puts 'Outbound LB - Ruby Demo'
  puts "Proxy: http://#{PROXY_HOST}:#{PROXY_PORT}"
  puts

  example_http_request
  example_https_request
  example_authenticated_proxy
  example_error_handling
  example_concurrent_requests
  example_environment_variables

  puts 'All examples completed!'
end

main if __FILE__ == $PROGRAM_NAME
