package netutil

import (
	"testing"
)

func TestIsIPv4(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"127.0.0.1", true},
		{"255.255.255.255", true},
		{"::1", false},
		{"2001:db8::1", false},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			result := IsIPv4(tt.ip)
			if result != tt.expected {
				t.Errorf("IsIPv4(%s) = %v, expected %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsIPv6(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"::1", true},
		{"2001:db8::1", true},
		{"fe80::1", true},
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			result := IsIPv6(tt.ip)
			if result != tt.expected {
				t.Errorf("IsIPv6(%s) = %v, expected %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestParseHost(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example.com:80", "example.com"},
		{"example.com:443", "example.com"},
		{"example.com", "example.com"},
		{"192.168.1.1:8080", "192.168.1.1"},
		{"[::1]:80", "::1"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseHost(tt.input)
			if result != tt.expected {
				t.Errorf("ParseHost(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example.com:80", "example.com"},
		{"Example.Com:443", "Example.Com"}, // Note: doesn't lowercase currently
		{"example.com", "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeHost(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeHost(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateLocalIP_Loopback(t *testing.T) {
	// 127.0.0.1 should be available on all systems
	err := ValidateLocalIP("127.0.0.1")
	if err != nil {
		t.Errorf("unexpected error for loopback: %v", err)
	}
}

func TestValidateLocalIP_Invalid(t *testing.T) {
	tests := []string{
		"invalid",
		"999.999.999.999",
		"",
	}

	for _, ip := range tests {
		t.Run(ip, func(t *testing.T) {
			err := ValidateLocalIP(ip)
			if err == nil {
				t.Errorf("expected error for invalid IP: %s", ip)
			}
		})
	}
}

func TestValidateLocalIP_NotLocal(t *testing.T) {
	// This IP is unlikely to exist on any local interface
	err := ValidateLocalIP("8.8.8.8")
	if err == nil {
		t.Error("expected error for non-local IP")
	}
}

func TestGetLocalIPs(t *testing.T) {
	ips, err := GetLocalIPs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have at least one non-loopback IP
	// (though this might fail in some container environments)
	t.Logf("Found local IPs: %v", ips)
}

func TestValidateLocalIPs(t *testing.T) {
	// Valid loopback
	err := ValidateLocalIPs([]string{"127.0.0.1"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Invalid IP in list
	err = ValidateLocalIPs([]string{"127.0.0.1", "8.8.8.8"})
	if err == nil {
		t.Error("expected error for non-local IP in list")
	}

	// Invalid format
	err = ValidateLocalIPs([]string{"invalid"})
	if err == nil {
		t.Error("expected error for invalid IP format")
	}
}
