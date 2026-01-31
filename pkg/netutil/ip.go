// Package netutil provides network utility functions.
package netutil

import (
	"fmt"
	"net"
)

// ValidateLocalIP checks if an IP address exists on a local interface.
func ValidateLocalIP(ipStr string) error {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %s", ipStr)
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return fmt.Errorf("getting interface addresses: %w", err)
	}

	for _, addr := range addrs {
		var ifIP net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ifIP = v.IP
		case *net.IPAddr:
			ifIP = v.IP
		}
		if ifIP != nil && ifIP.Equal(ip) {
			return nil
		}
	}

	return fmt.Errorf("IP %s not found on any local interface", ipStr)
}

// ValidateLocalIPs validates multiple IP addresses.
func ValidateLocalIPs(ips []string) error {
	for _, ip := range ips {
		if err := ValidateLocalIP(ip); err != nil {
			return err
		}
	}
	return nil
}

// GetLocalIPs returns all local IP addresses.
func GetLocalIPs() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("getting interface addresses: %w", err)
	}

	var ips []string
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip != nil && !ip.IsLoopback() {
			ips = append(ips, ip.String())
		}
	}
	return ips, nil
}

// IsIPv4 checks if a string is an IPv4 address.
func IsIPv4(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	return ip != nil && ip.To4() != nil
}

// IsIPv6 checks if a string is an IPv6 address.
func IsIPv6(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	return ip != nil && ip.To4() == nil
}

// ParseHost extracts the host from a host:port string.
func ParseHost(hostport string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		// Might not have a port
		return hostport
	}
	return host
}

// NormalizeHost returns a consistent host representation.
func NormalizeHost(host string) string {
	// Remove port if present
	h := ParseHost(host)
	// Lowercase for consistency
	return h
}
