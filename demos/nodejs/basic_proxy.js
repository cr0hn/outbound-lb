#!/usr/bin/env node
/**
 * Outbound LB - Node.js Demo
 *
 * Demonstrates how to use Outbound LB proxy with Node.js using axios and https-proxy-agent.
 * Includes examples for HTTP, HTTPS, authentication, error handling, and concurrent requests.
 *
 * Requirements:
 *     npm install axios https-proxy-agent http-proxy-agent
 *
 * Usage:
 *     node basic_proxy.js
 */

const http = require('http');
const https = require('https');

// Configuration from environment variables
const PROXY_HOST = process.env.PROXY_HOST || 'localhost';
const PROXY_PORT = process.env.PROXY_PORT || '3128';
const PROXY_USER = process.env.PROXY_USER || 'user';
const PROXY_PASS = process.env.PROXY_PASS || 'password';

const PROXY_URL = `http://${PROXY_HOST}:${PROXY_PORT}`;
const PROXY_URL_AUTH = `http://${PROXY_USER}:${PROXY_PASS}@${PROXY_HOST}:${PROXY_PORT}`;

// Try to load optional dependencies
let axios, HttpsProxyAgent, HttpProxyAgent;

try {
    axios = require('axios');
    HttpsProxyAgent = require('https-proxy-agent').HttpsProxyAgent;
    HttpProxyAgent = require('http-proxy-agent').HttpProxyAgent;
} catch (e) {
    console.error('Dependencies not installed. Run: npm install axios https-proxy-agent http-proxy-agent');
    process.exit(1);
}

/**
 * Example 1: Basic HTTP Request
 */
async function exampleHttpRequest() {
    console.log('='.repeat(60));
    console.log('Example 1: Basic HTTP Request');
    console.log('='.repeat(60));

    try {
        const agent = new HttpProxyAgent(PROXY_URL);
        const response = await axios.get('http://httpbin.org/ip', {
            httpAgent: agent,
            proxy: false, // Important: disable axios's default proxy handling
            timeout: 10000,
        });
        console.log(`Status: ${response.status}`);
        console.log(`Response: ${JSON.stringify(response.data)}`);
    } catch (error) {
        console.log(`Error: ${error.message}`);
    }

    console.log();
}

/**
 * Example 2: HTTPS Request (CONNECT tunnel)
 */
async function exampleHttpsRequest() {
    console.log('='.repeat(60));
    console.log('Example 2: HTTPS Request (CONNECT tunnel)');
    console.log('='.repeat(60));

    try {
        const agent = new HttpsProxyAgent(PROXY_URL);
        const response = await axios.get('https://httpbin.org/ip', {
            httpsAgent: agent,
            proxy: false,
            timeout: 10000,
        });
        console.log(`Status: ${response.status}`);
        console.log(`Response: ${JSON.stringify(response.data)}`);
    } catch (error) {
        console.log(`Error: ${error.message}`);
    }

    console.log();
}

/**
 * Example 3: Authenticated Proxy
 */
async function exampleAuthenticatedProxy() {
    console.log('='.repeat(60));
    console.log('Example 3: Authenticated Proxy');
    console.log('='.repeat(60));

    try {
        const agent = new HttpsProxyAgent(PROXY_URL_AUTH);
        const response = await axios.get('https://httpbin.org/ip', {
            httpsAgent: agent,
            proxy: false,
            timeout: 10000,
        });
        console.log(`Status: ${response.status}`);
        console.log(`Response: ${JSON.stringify(response.data)}`);
    } catch (error) {
        console.log(`Error: ${error.message}`);
    }

    console.log();
}

/**
 * Example 4: Error Handling
 */
async function exampleErrorHandling() {
    console.log('='.repeat(60));
    console.log('Example 4: Error Handling');
    console.log('='.repeat(60));

    // Test connection timeout
    console.log('Testing connection timeout...');
    try {
        const agent = new HttpsProxyAgent(PROXY_URL);
        await axios.get('http://httpbin.org/delay/5', {
            httpsAgent: agent,
            proxy: false,
            timeout: 2000, // Short timeout
        });
    } catch (error) {
        if (error.code === 'ECONNABORTED' || error.message.includes('timeout')) {
            console.log('  Caught timeout error (expected)');
        } else {
            console.log(`  Error: ${error.message}`);
        }
    }

    // Test invalid URL
    console.log('Testing invalid URL...');
    try {
        const agent = new HttpsProxyAgent(PROXY_URL);
        await axios.get('http://invalid.invalid.invalid', {
            httpsAgent: agent,
            proxy: false,
            timeout: 5000,
        });
    } catch (error) {
        console.log(`  Caught connection error (expected): ${error.code || error.message}`);
    }

    // Test proxy authentication failure
    console.log('Testing proxy auth failure (if auth required on proxy)...');
    try {
        const badProxyUrl = `http://wronguser:wrongpass@${PROXY_HOST}:${PROXY_PORT}`;
        const agent = new HttpsProxyAgent(badProxyUrl);
        const response = await axios.get('http://httpbin.org/ip', {
            httpsAgent: agent,
            httpAgent: new HttpProxyAgent(badProxyUrl),
            proxy: false,
            timeout: 5000,
            validateStatus: () => true, // Accept any status code
        });
        if (response.status === 407) {
            console.log('  Got 407 Proxy Authentication Required (expected)');
        } else {
            console.log(`  Status: ${response.status}`);
        }
    } catch (error) {
        console.log(`  Error: ${error.message}`);
    }

    console.log();
}

/**
 * Example 5: Concurrent Requests (Load Balancing Demo)
 */
async function exampleConcurrentRequests() {
    console.log('='.repeat(60));
    console.log('Example 5: Concurrent Requests (Load Balancing Demo)');
    console.log('='.repeat(60));

    const numRequests = 10;
    console.log(`Making ${numRequests} concurrent requests...`);

    const ipCounts = {};

    const makeRequest = async (requestId) => {
        try {
            const agent = new HttpProxyAgent(PROXY_URL);
            const response = await axios.get('http://httpbin.org/ip', {
                httpAgent: agent,
                proxy: false,
                timeout: 10000,
            });
            return { requestId, ip: response.data.origin };
        } catch (error) {
            return { requestId, ip: `error: ${error.message}` };
        }
    };

    const promises = [];
    for (let i = 0; i < numRequests; i++) {
        promises.push(makeRequest(i));
    }

    const results = await Promise.all(promises);

    for (const { requestId, ip } of results) {
        console.log(`  Request ${requestId}: ${ip}`);
        if (!ip.startsWith('error')) {
            ipCounts[ip] = (ipCounts[ip] || 0) + 1;
        }
    }

    console.log('\nIP Distribution:');
    for (const [ip, count] of Object.entries(ipCounts).sort()) {
        console.log(`  ${ip}: ${count} requests`);
    }

    console.log();
}

/**
 * Example 6: Using native HTTP module
 */
async function exampleNativeHttp() {
    console.log('='.repeat(60));
    console.log('Example 6: Using Native HTTP Module');
    console.log('='.repeat(60));

    return new Promise((resolve) => {
        const options = {
            hostname: PROXY_HOST,
            port: parseInt(PROXY_PORT),
            path: 'http://httpbin.org/ip',
            method: 'GET',
            headers: {
                'Host': 'httpbin.org',
            },
        };

        const req = http.request(options, (res) => {
            let data = '';
            res.on('data', (chunk) => {
                data += chunk;
            });
            res.on('end', () => {
                console.log(`Status: ${res.statusCode}`);
                try {
                    console.log(`Response: ${data}`);
                } catch (e) {
                    console.log(`Response: ${data}`);
                }
                console.log();
                resolve();
            });
        });

        req.on('error', (error) => {
            console.log(`Error: ${error.message}`);
            console.log();
            resolve();
        });

        req.setTimeout(10000, () => {
            req.destroy();
            console.log('Request timeout');
            console.log();
            resolve();
        });

        req.end();
    });
}

/**
 * Main function
 */
async function main() {
    console.log();
    console.log('Outbound LB - Node.js Demo');
    console.log(`Proxy: ${PROXY_URL}`);
    console.log();

    await exampleHttpRequest();
    await exampleHttpsRequest();
    await exampleAuthenticatedProxy();
    await exampleErrorHandling();
    await exampleConcurrentRequests();
    await exampleNativeHttp();

    console.log('All examples completed!');
}

main().catch(console.error);
